package caixa_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	caixausecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/caixa"
	"github.com/google/uuid"
)

type cenario struct {
	ctx                 context.Context
	aberturas           *caixausecases.AberturaUsecaseImpl
	extratos            *caixausecases.ExtratoUsecaseImpl
	saldos              *caixausecases.SaldoUsecaseImpl
	conciliacao         *caixausecases.ConciliacaoUsecaseImpl
	conciliacoesRepo    *repositorioConciliacoes
	empresasRepo        *repositorioEmpresas
	usuariosRepo        *repositorioUsuarios
	cargosRepo          *repositorioCargos
	contasRepo          *repositorioContas
	arquivosRepo        *repositorioArquivos
	storage             *armazenamento
	aberturasRepo       *repositorioAberturas
	extratosRepo        *repositorioExtratos
	saldosRepo          *repositorioSaldos
	mia                 uuid.UUID
	red                 uuid.UUID
	inativa             uuid.UUID
	sicrediMIA          uuid.UUID
	itauRed             uuid.UUID
	contaInativa        uuid.UUID
	contaEmpresaInativa uuid.UUID
	administrador       uuid.UUID
	financeiroMIA       uuid.UUID
	financeiroRed       uuid.UUID
	comum               uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	empresas := &repositorioEmpresas{itens: map[uuid.UUID]*domainempresa.Empresa{}, acessos: map[uuid.UUID]domainempresa.Acesso{}}
	usuarios := &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
	cargos := &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
	contas := &repositorioContas{itens: map[uuid.UUID]*domainfinanceiro.ContaBancaria{}}
	arquivos := &repositorioArquivos{itens: map[uuid.UUID]*domainsolicitacao.Arquivo{}, daSolicitacao: map[uuid.UUID]bool{}}
	storage := &armazenamento{conteudos: map[string][]byte{}}
	aberturas := &repositorioAberturas{itens: map[uuid.UUID]*domaincaixa.Abertura{}}
	extratos := &repositorioExtratos{contas: contas}
	saldos := &repositorioSaldos{contas: contas}
	conciliacoes := novoRepositorioConciliacoes()

	c := &cenario{
		ctx:              context.Background(),
		aberturas:        caixausecases.NewAberturaUsecase(aberturas, contas, empresas, usuarios, cargos),
		extratos:         caixausecases.NewExtratoUsecase(extratos, aberturas, arquivos, storage, contas, empresas, usuarios, cargos),
		saldos:           caixausecases.NewSaldoUsecase(saldos, empresas, usuarios, cargos),
		conciliacao:      caixausecases.NewConciliacaoUsecase(conciliacoes, contas, empresas, usuarios, cargos),
		conciliacoesRepo: conciliacoes,
		empresasRepo:     empresas,
		usuariosRepo:     usuarios,
		cargosRepo:       cargos,
		contasRepo:       contas,
		arquivosRepo:     arquivos,
		storage:          storage,
		aberturasRepo:    aberturas,
		extratosRepo:     extratos,
		saldosRepo:       saldos,
	}

	c.mia = c.novaEmpresa("MIA", true)
	c.red = c.novaEmpresa("Red Diamond", true)
	c.inativa = c.novaEmpresa("Gigante SP", false)
	c.sicrediMIA = c.novaConta(t, c.mia, domainfinanceiro.Sicredi, "0812", "54321", "0", true)
	c.itauRed = c.novaConta(t, c.red, domainfinanceiro.Itau, "1234", "98765", "4", true)
	c.contaInativa = c.novaConta(t, c.mia, domainfinanceiro.Bradesco, "1", "111", "", false)
	c.contaEmpresaInativa = c.novaConta(t, c.inativa, domainfinanceiro.Inter, "1", "222", "", true)
	c.administrador = c.novoUsuario("Ana Admin", true, false, domainempresa.AcessoTotal())
	c.financeiroMIA = c.novoUsuario("Fábio MIA", false, true, domainempresa.AcessoRestrito([]uuid.UUID{c.mia, c.inativa}))
	c.financeiroRed = c.novoUsuario("Rita Red", false, true, domainempresa.AcessoRestrito([]uuid.UUID{c.red}))
	c.comum = c.novoUsuario("Carla Comum", false, false, domainempresa.AcessoTotal())

	return c
}

func (c *cenario) novaEmpresa(nome string, ativa bool) uuid.UUID {
	id := uuid.New()
	c.empresasRepo.itens[id] = &domainempresa.Empresa{ID: id, NomeFantasia: nome, Ativa: ativa}

	return id
}

func (c *cenario) novaConta(t *testing.T, empresaID uuid.UUID, banco domainfinanceiro.Banco, agencia, numero, digito string, ativa bool) uuid.UUID {
	t.Helper()

	numeroAgencia, err := domainfinanceiro.NovaAgencia(agencia, nil)
	if err != nil {
		t.Fatalf("agência inválida: %v", err)
	}
	var digitoConta *string
	if digito != "" {
		digitoConta = &digito
	}
	numeroConta, err := domainfinanceiro.NovaConta(numero, digitoConta)
	if err != nil {
		t.Fatalf("conta inválida: %v", err)
	}

	conta, err := domainfinanceiro.NovaContaBancaria(domainfinanceiro.DadosContaBancaria{
		EmpresaID: empresaID, Banco: banco, Tipo: domainfinanceiro.TipoContaCorrente,
		Agencia: numeroAgencia, Conta: numeroConta, Apelido: "Conta " + numero,
	})
	if err != nil {
		t.Fatalf("conta bancária inválida: %v", err)
	}
	conta.Ativa = ativa
	c.contasRepo.itens[conta.ID] = conta

	return conta.ID
}

func (c *cenario) novoUsuario(nome string, administrador, financeiro bool, acesso domainempresa.Acesso) uuid.UUID {
	cargoID := uuid.New()
	c.cargosRepo.itens[cargoID] = &domainusuarios.Cargo{
		ID: cargoID, Nome: "Cargo de " + nome, Ativo: true, Administrador: administrador, Financeiro: financeiro,
	}

	usuarioID := uuid.New()
	c.usuariosRepo.itens[usuarioID] = &domainusuarios.Usuario{ID: usuarioID, Nome: nome, CargoID: cargoID, Ativo: true}
	c.empresasRepo.acessos[usuarioID] = acesso

	return usuarioID
}

func (c *cenario) novoArquivo(dono uuid.UUID, contentType, conteudo string) uuid.UUID {
	id := uuid.New()
	chave := dono.String() + "/" + id.String() + "/extrato.ofx"
	c.arquivosRepo.itens[id] = &domainsolicitacao.Arquivo{
		ID: id, ProprietarioID: dono, Nome: "extrato.ofx", Chave: chave, ContentType: contentType, Tamanho: int64(len(conteudo)),
	}
	c.storage.conteudos[chave] = []byte(conteudo)

	return id
}

func (c *cenario) abrirConta(t *testing.T, contaID uuid.UUID, diasAtras int, saldo int64) *domaincaixa.Abertura {
	t.Helper()

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: contaID,
		DataCorte:       dia(-diasAtras),
		SaldoCentavos:   saldo,
		UsuarioID:       c.administrador,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	c.aberturasRepo.itens[contaID] = abertura

	return abertura
}

func dia(deslocamento int) time.Time {
	return domaincaixa.Hoje(time.Now().UTC()).AddDate(0, 0, deslocamento)
}

func dataOFX(deslocamento int) string {
	return dia(deslocamento).Format("20060102")
}

func extratoOFX(banco, agencia, conta string, transacoes ...string) string {
	return `OFXHEADER:100
DATA:OFXSGML

<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS>
<BANKACCTFROM><BANKID>` + banco + `<BRANCHID>` + agencia + `<ACCTID>` + conta + `</BANKACCTFROM>
<BANKTRANLIST>
` + strings.Join(transacoes, "\n") + `
</BANKTRANLIST>
<LEDGERBAL><BALAMT>1000.00<DTASOF>` + dataOFX(-1) + `</LEDGERBAL>
</STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`
}

func transacaoOFX(fitid string, deslocamento int, valor, memo string) string {
	return "<STMTTRN><DTPOSTED>" + dataOFX(deslocamento) + "<TRNAMT>" + valor + "<FITID>" + fitid + "<MEMO>" + memo + "</STMTTRN>"
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func inteiro(valor int) *int {
	return &valor
}

func timePtr(valor time.Time) *time.Time {
	return &valor
}
