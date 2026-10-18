package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

var (
	_ portsreceber.EmpresaRepository             = (*postgres.EmpresaRepository)(nil)
	_ portsreceber.ClienteRepository             = (*postgres.ClienteRepository)(nil)
	_ portsreceber.ContaBancariaRepository       = (*postgres.ContaBancariaRepository)(nil)
	_ portsreceber.CategoriaFinanceiraRepository = (*postgres.CategoriaFinanceiraRepository)(nil)
	_ portsreceber.UsuarioRepository             = (*postgres.UsuarioRepository)(nil)
	_ portsreceber.CargoRepository               = (*postgres.CargoRepository)(nil)
)

var agoraDoTitulo = time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC)

type cenarioBancoTitulos struct {
	*cenarioBancoClassificacao
	ctx       context.Context
	titulos   *postgres.TituloReceberRepository
	clienteID uuid.UUID
	contaID   uuid.UUID
}

func novoCenarioTitulos(t *testing.T) *cenarioBancoTitulos {
	t.Helper()

	base := cenarioClassificacao(t)
	c := &cenarioBancoTitulos{
		cenarioBancoClassificacao: base,
		ctx:                       context.Background(),
		titulos:                   postgres.NewTituloReceberRepository(base.banco),
	}
	c.clienteID = c.inserirCliente(t, "11.444.777/0001-61", "Construtora Alfa")
	c.contaID = c.inserirConta(t, c.empresaID, "12345", "Cobrança")

	return c
}

func (c *cenarioBancoTitulos) inserirCliente(t *testing.T, documento, nome string) uuid.UUID {
	t.Helper()

	valor, err := domaincliente.NovoDocumento(documento)
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	cliente, err := domaincliente.NovoCliente(valor, domaincliente.DadosCliente{
		Nome:   nome,
		Tipo:   domaincliente.TipoClienteConstrutora,
		UF:     domainempresa.UFDe("PB"),
		Cidade: "João Pessoa",
	}, nil)
	if err != nil {
		t.Fatalf("cliente de teste inválido: %v", err)
	}
	if _, err := postgres.NewClienteRepository(c.banco).Criar(c.ctx, cliente); err != nil {
		t.Fatalf("criação do cliente falhou: %v", err)
	}

	return cliente.ID
}

func (c *cenarioBancoTitulos) tituloDeDominio(t *testing.T, numero string, vencimento time.Time) *domainreceber.Titulo {
	t.Helper()

	titulo, err := domainreceber.NovoTituloManual(c.empresaID, c.solicitanteID, domainreceber.DadosTitulo{
		DadosEdicao: domainreceber.DadosEdicao{
			Numero:                numero,
			Parcela:               1,
			Vencimento:            vencimento,
			ValorCentavos:         150000,
			CategoriaFinanceiraID: &c.receitaID,
		},
		ClienteID: c.clienteID,
	}, agoraDoTitulo)
	if err != nil {
		t.Fatalf("título de teste inválido: %v", err)
	}

	return titulo
}

func (c *cenarioBancoTitulos) criar(titulo *domainreceber.Titulo) error {
	return c.titulos.Criar(c.ctx, titulo, domainreceber.NovoHistorico(
		titulo.ID, titulo.UsuarioID, domainreceber.EventoCriado, nil, titulo.Status, "", titulo.CriadoEm,
	))
}

func (c *cenarioBancoTitulos) novoTitulo(t *testing.T, numero string, vencimento time.Time) *domainreceber.Titulo {
	t.Helper()

	titulo := c.tituloDeDominio(t, numero, vencimento)
	if err := c.criar(titulo); err != nil {
		t.Fatalf("criação do título falhou: %v", err)
	}

	return titulo
}

func (c *cenarioBancoTitulos) obterTitulo(t *testing.T, id uuid.UUID) *domainreceber.Titulo {
	t.Helper()

	titulo, err := c.titulos.Obter(c.ctx, id)
	if err != nil || titulo == nil {
		t.Fatalf("Obter(%s) = %v, %v", id, titulo, err)
	}

	return titulo
}

func (c *cenarioBancoTitulos) listar(t *testing.T, filtro portsreceber.TituloFiltro) []uuid.UUID {
	t.Helper()

	titulos, err := c.titulos.Listar(c.ctx, filtro)
	if err != nil {
		t.Fatalf("Listar falhou: %v", err)
	}

	ids := make([]uuid.UUID, 0, len(titulos))
	for _, titulo := range titulos {
		ids = append(ids, titulo.ID)
	}

	return ids
}

func diaDoTitulo(mes time.Month, dia int) time.Time {
	return time.Date(2026, mes, dia, 0, 0, 0, 0, time.UTC)
}

func TestTituloReceberCriarEObterPreservaOsDados(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.tituloDeDominio(t, "ALG-10", diaDoTitulo(11, 10))
	titulo.ContaBancariaID = &c.contaID
	titulo.Status = domainreceber.StatusEmBanco
	titulo.Observacao = "aluguel do galpão"

	if err := c.criar(titulo); err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	obtido := c.obterTitulo(t, titulo.ID)
	if obtido.Numero != "ALG-10" || obtido.Status != domainreceber.StatusEmBanco || *obtido.ContaBancariaID != c.contaID {
		t.Fatalf("título = %+v", obtido)
	}
	if !obtido.Vencimento.Equal(diaDoTitulo(11, 10)) || obtido.Competencia.String() != "2026-11" || obtido.Versao != 1 {
		t.Fatalf("datas ou versão = %s, %s, %d", obtido.Vencimento, obtido.Competencia, obtido.Versao)
	}
	if *obtido.CategoriaFinanceiraID != c.receitaID || obtido.Origem != domainreceber.OrigemManual || obtido.Observacao != "aluguel do galpão" {
		t.Fatalf("classificação = %+v", obtido)
	}

	historico, err := c.titulos.ListarHistorico(c.ctx, titulo.ID)
	if err != nil || len(historico) != 1 || historico[0].Evento != domainreceber.EventoCriado || historico[0].DeStatus != nil {
		t.Fatalf("histórico = %+v, %v", historico, err)
	}

	if ausente, err := c.titulos.Obter(c.ctx, uuid.New()); ausente != nil || err != nil {
		t.Fatalf("Obter de id ausente = %v, %v", ausente, err)
	}
}

func TestTituloReceberCriarConfereAsDependenciasNaTransacao(t *testing.T) {
	c := novoCenarioTitulos(t)

	despesa := c.tituloDeDominio(t, "1", diaDoTitulo(11, 10))
	despesa.CategoriaFinanceiraID = &c.despesaID
	esperarErro(t, c.criar(despesa), domain.ErrValidacao, "deve ser de receita")

	raiz := c.tituloDeDominio(t, "2", diaDoTitulo(11, 10))
	raiz.CategoriaFinanceiraID = &c.raizID
	esperarErro(t, c.criar(raiz), domain.ErrValidacao, "subcategoria ativa")

	outraEmpresa := inserirEmpresa(t, c.banco, "45.723.174/0001-10", "Red Diamond")
	contaDeOutra := c.inserirConta(t, outraEmpresa, "77777", "Outra")
	deOutra := c.tituloDeDominio(t, "3", diaDoTitulo(11, 10))
	deOutra.ContaBancariaID, deOutra.Status = &contaDeOutra, domainreceber.StatusEmBanco
	esperarErro(t, c.criar(deOutra), domain.ErrValidacao, "mesma empresa")

	if _, err := c.banco.Exec(`UPDATE cliente SET ativo = false WHERE id = $1`, c.clienteID); err != nil {
		t.Fatal(err)
	}
	esperarErro(t, c.criar(c.tituloDeDominio(t, "4", diaDoTitulo(11, 10))), domain.ErrValidacao, "cliente inativo")

	if _, err := c.banco.Exec(`UPDATE cliente SET ativo = true WHERE id = $1`, c.clienteID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.banco.Exec(`UPDATE empresa SET ativa = false WHERE id = $1`, c.empresaID); err != nil {
		t.Fatal(err)
	}
	esperarErro(t, c.criar(c.tituloDeDominio(t, "5", diaDoTitulo(11, 10))), domain.ErrValidacao, "empresa inativa")
}

func TestTituloReceberNumeroRepetidoNaEmpresaE409(t *testing.T) {
	c := novoCenarioTitulos(t)
	primeiro := c.novoTitulo(t, "NF-100", diaDoTitulo(11, 10))

	esperarErro(t, c.criar(c.tituloDeDominio(t, "nf-100", diaDoTitulo(11, 20))), domain.ErrConflito, "número e parcela")

	if err := primeiro.Cancelar("lançado errado", agoraDoTitulo); err != nil {
		t.Fatal(err)
	}
	if err := c.titulos.Atualizar(c.ctx, &portsreceber.Alteracao{
		Titulo: primeiro, VersaoLida: 1, StatusAnterior: domainreceber.StatusCarteira,
	}); err != nil {
		t.Fatalf("cancelar falhou: %v", err)
	}
	if err := c.criar(c.tituloDeDominio(t, "NF-100", diaDoTitulo(11, 20))); err != nil {
		t.Fatalf("o número de título cancelado pode ser reaproveitado: %v", err)
	}
}

func TestTituloReceberAtualizarGuardaStatusEVersao(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "ALG-10", diaDoTitulo(11, 10))

	if err := titulo.EnviarAoBanco(c.contaID, agoraDoTitulo); err != nil {
		t.Fatal(err)
	}
	alteracao := &portsreceber.Alteracao{
		Titulo:         titulo,
		VersaoLida:     1,
		StatusAnterior: domainreceber.StatusCarteira,
		ConferirConta:  true,
		Historico: domainreceber.NovoHistorico(titulo.ID, c.solicitanteID, domainreceber.EventoEnviadoBanco,
			statusDe(domainreceber.StatusCarteira), domainreceber.StatusEmBanco, "", agoraDoTitulo),
	}
	if err := c.titulos.Atualizar(c.ctx, alteracao); err != nil {
		t.Fatalf("Atualizar falhou: %v", err)
	}
	if titulo.Versao != 2 {
		t.Fatalf("versão em memória = %d", titulo.Versao)
	}

	obtido := c.obterTitulo(t, titulo.ID)
	if obtido.Status != domainreceber.StatusEmBanco || *obtido.ContaBancariaID != c.contaID || obtido.Versao != 2 {
		t.Fatalf("título = %+v", obtido)
	}

	if err := c.titulos.Atualizar(c.ctx, alteracao); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("versão antiga = %v", err)
	}

	ausente := c.tituloDeDominio(t, "X", diaDoTitulo(11, 10))
	if err := c.titulos.Atualizar(c.ctx, &portsreceber.Alteracao{
		Titulo: ausente, VersaoLida: 1, StatusAnterior: domainreceber.StatusCarteira,
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("título ausente = %v", err)
	}

	historico, err := c.titulos.ListarHistorico(c.ctx, titulo.ID)
	if err != nil || len(historico) != 2 || historico[1].Evento != domainreceber.EventoEnviadoBanco || *historico[1].DeStatus != domainreceber.StatusCarteira {
		t.Fatalf("histórico = %+v, %v", historico, err)
	}
}

func TestTituloReceberAtualizarConfereContaInativa(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "ALG-10", diaDoTitulo(11, 10))
	if _, err := c.banco.Exec(`UPDATE conta_bancaria SET ativa = false WHERE id = $1`, c.contaID); err != nil {
		t.Fatal(err)
	}

	if err := titulo.EnviarAoBanco(c.contaID, agoraDoTitulo); err != nil {
		t.Fatal(err)
	}
	err := c.titulos.Atualizar(c.ctx, &portsreceber.Alteracao{
		Titulo: titulo, VersaoLida: 1, StatusAnterior: domainreceber.StatusCarteira, ConferirConta: true,
	})
	esperarErro(t, err, domain.ErrValidacao, "conta bancária inativa")
}

func TestTituloReceberListarFiltra(t *testing.T) {
	c := novoCenarioTitulos(t)
	novembro := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	outubro := c.novoTitulo(t, "2", diaDoTitulo(10, 20))
	dezembro := c.novoTitulo(t, "3", diaDoTitulo(12, 5))
	if err := dezembro.EnviarAoBanco(c.contaID, agoraDoTitulo); err != nil {
		t.Fatal(err)
	}
	if err := c.titulos.Atualizar(c.ctx, &portsreceber.Alteracao{
		Titulo: dezembro, VersaoLida: 1, StatusAnterior: domainreceber.StatusCarteira,
	}); err != nil {
		t.Fatal(err)
	}

	outraEmpresa := inserirEmpresa(t, c.banco, "45.723.174/0001-10", "Red Diamond")
	daOutra := c.tituloDeDominio(t, "1", diaDoTitulo(11, 1))
	daOutra.EmpresaID = outraEmpresa
	if err := c.criar(daOutra); err != nil {
		t.Fatal(err)
	}

	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{}), outubro.ID, daOutra.ID, novembro.ID, dezembro.ID)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{EmpresaIDs: []uuid.UUID{c.empresaID}}), outubro.ID, novembro.ID, dezembro.ID)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{EmpresaIDs: []uuid.UUID{}}))
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{Empresa: &outraEmpresa}), daOutra.ID)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{Status: domainreceber.StatusEmBanco}), dezembro.ID)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{ContaBancariaID: &c.contaID}), dezembro.ID)
	de, ate := diaDoTitulo(11, 1), diaDoTitulo(11, 30)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{VencimentoDe: &de, VencimentoAte: &ate}), daOutra.ID, novembro.ID)
	hoje := diaDoTitulo(11, 5)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{VencidosEm: &hoje}), outubro.ID, daOutra.ID)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{Origem: domainreceber.OrigemPedido}))
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{OrigemCancelada: true}))
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{ClienteID: &c.clienteID, PaginacaoFiltro: domain.PaginacaoFiltro{Page: 2, Size: 3}}), dezembro.ID)
}

func statusDe(status domainreceber.Status) *domainreceber.Status {
	return &status
}

func esperarIDs(t *testing.T, obtidos []uuid.UUID, esperados ...uuid.UUID) {
	t.Helper()

	if len(obtidos) != len(esperados) {
		t.Fatalf("ids = %v; esperados %v", obtidos, esperados)
	}
	for i := range esperados {
		if obtidos[i] != esperados[i] {
			t.Fatalf("ids = %v; esperados %v", obtidos, esperados)
		}
	}
}
