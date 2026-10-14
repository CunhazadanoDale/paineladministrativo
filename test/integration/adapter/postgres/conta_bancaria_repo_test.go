package postgres_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioBancoContas struct {
	ctx    context.Context
	banco  *sqlx.DB
	contas *postgres.ContaBancariaRepository
	mia    uuid.UUID
	red    uuid.UUID
}

func novoCenarioContas(t *testing.T) *cenarioBancoContas {
	t.Helper()

	banco := helpers.BancoDoTeste(t)
	empresas := postgres.NewEmpresaRepository(banco)

	return &cenarioBancoContas{
		ctx:    context.Background(),
		banco:  banco,
		contas: postgres.NewContaBancariaRepository(banco),
		mia:    novaEmpresaNoBanco(t, empresas, "11.222.333/0001-81", "MIA"),
		red:    novaEmpresaNoBanco(t, empresas, "33.400.004/0001-16", "Red Diamond"),
	}
}

func (c *cenarioBancoContas) novaConta(t *testing.T, empresaID uuid.UUID, banco domainfinanceiro.Banco, agencia, numero, apelido string) *domainfinanceiro.ContaBancaria {
	t.Helper()

	conta := contaDeDominio(t, empresaID, banco, agencia, numero, apelido)
	if err := c.contas.Criar(c.ctx, conta); err != nil {
		t.Fatalf("criação da conta %q falhou: %v", apelido, err)
	}

	return conta
}

func (c *cenarioBancoContas) desativarEmpresa(t *testing.T, empresaID uuid.UUID) {
	t.Helper()

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE empresa SET ativa = FALSE WHERE id = $1`, empresaID); err != nil {
		t.Fatalf("desativar a empresa falhou: %v", err)
	}
}

func (c *cenarioBancoContas) obter(t *testing.T, id uuid.UUID) *domainfinanceiro.ContaBancaria {
	t.Helper()

	conta, err := c.contas.Obter(c.ctx, id)
	if err != nil || conta == nil {
		t.Fatalf("Obter(%s) = %v, %v", id, conta, err)
	}

	return conta
}

func contaDeDominio(t *testing.T, empresaID uuid.UUID, banco domainfinanceiro.Banco, agencia, numero, apelido string) *domainfinanceiro.ContaBancaria {
	t.Helper()

	numeroAgencia, err := domainfinanceiro.NovaAgencia(agencia, nil)
	if err != nil {
		t.Fatalf("agência de teste inválida: %v", err)
	}
	numeroConta, err := domainfinanceiro.NovaConta(numero, nil)
	if err != nil {
		t.Fatalf("conta de teste inválida: %v", err)
	}
	conta, err := domainfinanceiro.NovaContaBancaria(domainfinanceiro.DadosContaBancaria{
		EmpresaID: empresaID,
		Banco:     banco,
		Tipo:      domainfinanceiro.TipoContaCorrente,
		Agencia:   numeroAgencia,
		Conta:     numeroConta,
		Apelido:   apelido,
	})
	if err != nil {
		t.Fatalf("conta de domínio inválida: %v", err)
	}

	return conta
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func TestContaBancariaCriarEObterPreservaOsDados(t *testing.T) {
	c := novoCenarioContas(t)
	criada := c.novaConta(t, c.mia, domainfinanceiro.Cora, "0001", "12345-6", "Cora pagamentos")
	semDigito := c.novaConta(t, c.mia, domainfinanceiro.Itau, "1234", "9876", "Itaú")

	conta := c.obter(t, criada.ID)
	if conta.EmpresaID != c.mia || conta.Banco != domainfinanceiro.Cora || conta.Tipo != domainfinanceiro.TipoContaCorrente {
		t.Errorf("conta lida = %+v", conta)
	}
	if conta.Agencia.Formatado() != "0001" || conta.Conta.Formatado() != "12345-6" || conta.Apelido != "Cora pagamentos" {
		t.Errorf("dados bancários = %q %q %q", conta.Agencia.Formatado(), conta.Conta.Formatado(), conta.Apelido)
	}
	if !conta.Ativa || conta.Versao != 1 || conta.CriadoEm.Sub(criada.CriadoEm).Abs() > time.Microsecond {
		t.Errorf("estado = ativa %v versão %d criada %v", conta.Ativa, conta.Versao, conta.CriadoEm)
	}

	if lida := c.obter(t, semDigito.ID); lida.Agencia.Digito() != nil || lida.Conta.Digito() != nil {
		t.Errorf("dígitos ausentes voltaram como %v/%v", lida.Agencia.Digito(), lida.Conta.Digito())
	}

	inexistente, err := c.contas.Obter(c.ctx, uuid.New())
	if err != nil || inexistente != nil {
		t.Errorf("conta inexistente = %v, %v", inexistente, err)
	}
}

func TestContaBancariaNumeroUnicoNoGrupoIgnorandoZerosEDigito(t *testing.T) {
	c := novoCenarioContas(t)
	c.novaConta(t, c.mia, domainfinanceiro.Cora, "0001", "12345-6", "Cora")

	repetidas := []*domainfinanceiro.ContaBancaria{
		contaDeDominio(t, c.mia, domainfinanceiro.Cora, "1", "0012345", "Outra"),
		contaDeDominio(t, c.red, domainfinanceiro.Cora, "001", "12345-9", "Cora da Red"),
	}
	for _, conta := range repetidas {
		esperarErro(t, c.contas.Criar(c.ctx, conta), domain.ErrConflito, "já existe uma conta com este banco, agência e número")
	}

	c.novaConta(t, c.mia, domainfinanceiro.Inter, "0001", "12345", "Inter")
	c.novaConta(t, c.mia, domainfinanceiro.Cora, "0002", "12345", "Cora agência 2")
}

func TestContaBancariaApelidoUnicoPorEmpresa(t *testing.T) {
	c := novoCenarioContas(t)
	c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "100", "Movimento")

	repetida := contaDeDominio(t, c.mia, domainfinanceiro.Itau, "1", "200", "MOVIMENTO")
	esperarErro(t, c.contas.Criar(c.ctx, repetida), domain.ErrConflito, "já existe uma conta com este apelido na empresa")

	c.novaConta(t, c.red, domainfinanceiro.Itau, "1", "200", "Movimento")
}

func TestContaBancariaCriarExigeEmpresaAtivaEExistente(t *testing.T) {
	c := novoCenarioContas(t)
	c.desativarEmpresa(t, c.red)

	esperarErro(t, c.contas.Criar(c.ctx, contaDeDominio(t, c.red, domainfinanceiro.Cora, "1", "1", "A")),
		domain.ErrValidacao, "empresa inativa não recebe conta bancária")
	esperarErro(t, c.contas.Criar(c.ctx, contaDeDominio(t, uuid.New(), domainfinanceiro.Cora, "1", "1", "A")),
		domain.ErrValidacao, "empresa da conta bancária não encontrada")
}

func TestContaBancariaListarFiltraEOrdena(t *testing.T) {
	c := novoCenarioContas(t)
	zeta := c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "zeta")
	alfa := c.novaConta(t, c.mia, domainfinanceiro.Itau, "1", "2", "Alfa")
	beta := c.novaConta(t, c.red, domainfinanceiro.Itau, "1", "3", "beta")
	inativa := c.novaConta(t, c.red, domainfinanceiro.Sicredi, "1", "4", "Gama")
	inativa.Desativar(time.Now().UTC())
	if err := c.contas.Desativar(c.ctx, inativa); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}

	itau := domainfinanceiro.Itau
	poupanca := domainfinanceiro.TipoContaPoupanca
	ativa := true
	casos := map[string]struct {
		filtro    portsout.ContaBancariaFiltro
		esperadas []*domainfinanceiro.ContaBancaria
	}{
		"todas por apelido":    {portsout.ContaBancariaFiltro{}, []*domainfinanceiro.ContaBancaria{alfa, beta, inativa, zeta}},
		"empresas do acesso":   {portsout.ContaBancariaFiltro{EmpresaIDs: []uuid.UUID{c.red}}, []*domainfinanceiro.ContaBancaria{beta, inativa}},
		"acesso vazio":         {portsout.ContaBancariaFiltro{EmpresaIDs: []uuid.UUID{}}, nil},
		"banco":                {portsout.ContaBancariaFiltro{Banco: &itau}, []*domainfinanceiro.ContaBancaria{alfa, beta}},
		"tipo":                 {portsout.ContaBancariaFiltro{Tipo: &poupanca}, nil},
		"ativas":               {portsout.ContaBancariaFiltro{Ativa: &ativa}, []*domainfinanceiro.ContaBancaria{alfa, beta, zeta}},
		"combinado com página": {portsout.ContaBancariaFiltro{Ativa: &ativa, PaginacaoFiltro: domain.PaginacaoFiltro{Page: 2, Size: 2}}, []*domainfinanceiro.ContaBancaria{zeta}},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			contas, err := c.contas.Listar(c.ctx, caso.filtro)
			if err != nil {
				t.Fatalf("Listar falhou: %v", err)
			}
			if len(contas) != len(caso.esperadas) {
				t.Fatalf("Listar devolveu %d contas, esperava %d", len(contas), len(caso.esperadas))
			}
			for i, conta := range contas {
				if conta.ID != caso.esperadas[i].ID {
					t.Errorf("posição %d = %q, esperava %q", i, conta.Apelido, caso.esperadas[i].Apelido)
				}
			}
		})
	}
}

func TestContaBancariaAtualizarAvancaVersao(t *testing.T) {
	c := novoCenarioContas(t)
	conta := c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "Antigo")

	if err := conta.AlterarApelido("Novo", time.Now().UTC()); err != nil {
		t.Fatalf("AlterarApelido falhou: %v", err)
	}
	if err := c.contas.Atualizar(c.ctx, conta); err != nil {
		t.Fatalf("Atualizar falhou: %v", err)
	}
	if conta.Versao != 2 {
		t.Errorf("versão em memória = %d", conta.Versao)
	}
	if lida := c.obter(t, conta.ID); lida.Apelido != "Novo" || lida.Versao != 2 {
		t.Errorf("conta lida = %q versão %d", lida.Apelido, lida.Versao)
	}

	antiga := *conta
	antiga.Versao = 1
	esperarErro(t, c.contas.Atualizar(c.ctx, &antiga), domain.ErrConflito, "alterada por outra pessoa")

	sumida := *conta
	sumida.ID = uuid.New()
	esperarErro(t, c.contas.Atualizar(c.ctx, &sumida), domain.ErrNotFound, "conta bancária não encontrada")
}

func TestContaBancariaAtualizarApelidoRepetidoNaEmpresa(t *testing.T) {
	c := novoCenarioContas(t)
	c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "Movimento")
	outra := c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "2", "Reserva")

	if err := outra.AlterarApelido("movimento", time.Now().UTC()); err != nil {
		t.Fatalf("AlterarApelido falhou: %v", err)
	}
	esperarErro(t, c.contas.Atualizar(c.ctx, outra), domain.ErrConflito, "já existe uma conta com este apelido na empresa")
}

func TestContaBancariaAtivarExigeEmpresaAtiva(t *testing.T) {
	c := novoCenarioContas(t)
	conta := c.novaConta(t, c.red, domainfinanceiro.Cora, "1", "1", "Red")

	conta.Desativar(time.Now().UTC())
	if err := c.contas.Desativar(c.ctx, conta); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}
	c.desativarEmpresa(t, c.red)

	conta.Ativar(time.Now().UTC())
	esperarErro(t, c.contas.Ativar(c.ctx, conta), domain.ErrValidacao, "empresa inativa não recebe conta bancária")
	if lida := c.obter(t, conta.ID); lida.Ativa || lida.Versao != 2 {
		t.Errorf("conta lida = ativa %v versão %d", lida.Ativa, lida.Versao)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE empresa SET ativa = TRUE WHERE id = $1`, c.red); err != nil {
		t.Fatalf("reativar a empresa falhou: %v", err)
	}
	if err := c.contas.Ativar(c.ctx, conta); err != nil {
		t.Fatalf("Ativar falhou: %v", err)
	}
	if lida := c.obter(t, conta.ID); !lida.Ativa || lida.Versao != 3 || conta.Versao != 3 {
		t.Errorf("conta lida = ativa %v versão %d/%d", lida.Ativa, lida.Versao, conta.Versao)
	}
}

func TestContaBancariaAtivarComVersaoAntiga(t *testing.T) {
	c := novoCenarioContas(t)
	conta := c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "MIA")
	antiga := *conta

	conta.Desativar(time.Now().UTC())
	if err := c.contas.Desativar(c.ctx, conta); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}

	esperarErro(t, c.contas.Desativar(c.ctx, &antiga), domain.ErrConflito, "alterada por outra pessoa")
	esperarErro(t, c.contas.Ativar(c.ctx, &antiga), domain.ErrConflito, "alterada por outra pessoa")
}

func TestContaBancariaEdicoesSimultaneasSoUmaGrava(t *testing.T) {
	c := novoCenarioContas(t)
	original := c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "Original")

	const concorrentes = 8
	var (
		espera    sync.WaitGroup
		trava     sync.Mutex
		gravadas  int
		conflitos int
		outros    []error
	)
	for i := range concorrentes {
		espera.Add(1)
		go func() {
			defer espera.Done()

			copia := *original
			err := copia.AlterarApelido("Apelido "+string(rune('A'+i)), time.Now().UTC())
			if err == nil {
				err = c.contas.Atualizar(c.ctx, &copia)
			}

			trava.Lock()
			defer trava.Unlock()
			switch {
			case err == nil:
				gravadas++
			case errors.Is(err, domain.ErrConflito):
				conflitos++
			default:
				outros = append(outros, err)
			}
		}()
	}
	espera.Wait()

	if len(outros) > 0 {
		t.Fatalf("erros inesperados: %v", outros)
	}
	if gravadas != 1 || conflitos != concorrentes-1 {
		t.Fatalf("%d gravadas e %d conflitos", gravadas, conflitos)
	}
	if lida := c.obter(t, original.ID); lida.Versao != 2 {
		t.Errorf("versão final = %d", lida.Versao)
	}
}

func TestContaBancariaRemover(t *testing.T) {
	c := novoCenarioContas(t)
	conta := c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "MIA")

	if err := c.contas.Remover(c.ctx, conta.ID); err != nil {
		t.Fatalf("Remover falhou: %v", err)
	}
	if lida, err := c.contas.Obter(c.ctx, conta.ID); err != nil || lida != nil {
		t.Errorf("conta removida = %v, %v", lida, err)
	}

	esperarErro(t, c.contas.Remover(c.ctx, conta.ID), domain.ErrNotFound, "conta bancária não encontrada")
}

func TestContaBancariaPrendeAEmpresa(t *testing.T) {
	c := novoCenarioContas(t)
	c.novaConta(t, c.mia, domainfinanceiro.Cora, "1", "1", "MIA")

	_, err := postgres.NewEmpresaRepository(c.banco).Remover(c.ctx, c.mia)
	esperarErro(t, err, domain.ErrValidacao, "registro em uso")
}
