package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioBancoSolicitacoes struct {
	banco         *sqlx.DB
	solicitacoes  *postgres.SolicitacaoRepository
	arquivos      *postgres.ArquivoRepository
	aprovadores   *postgres.AprovadorRepository
	usuarios      *postgres.UsuarioRepository
	cargos        *postgres.CargoRepository
	solicitanteID uuid.UUID
	terceiroID    uuid.UUID
}

func cenarioSolicitacoes(t *testing.T) *cenarioBancoSolicitacoes {
	t.Helper()

	banco := helpers.BancoDoTeste(t)
	cargos := postgres.NewCargoRepository(banco)
	usuarios := postgres.NewUsuarioRepository(banco)

	return &cenarioBancoSolicitacoes{
		banco:         banco,
		solicitacoes:  postgres.NewSolicitacaoRepository(banco),
		arquivos:      postgres.NewArquivoRepository(banco),
		aprovadores:   postgres.NewAprovadorRepository(banco),
		usuarios:      usuarios,
		cargos:        cargos,
		solicitanteID: inserirUsuario(t, cargos, usuarios, "Ana Souza", "ana.souza@exemplo.com"),
		terceiroID:    inserirUsuario(t, cargos, usuarios, "Bruno Lima", "bruno.lima@exemplo.com"),
	}
}

func inserirUsuario(t *testing.T, cargos *postgres.CargoRepository, usuarios *postgres.UsuarioRepository, nome, email string) uuid.UUID {
	t.Helper()

	cargoID, err := cargos.Create(context.Background(), novoCargo("Cargo de "+nome))
	if err != nil {
		t.Fatalf("não criei o cargo do cenário: %v", err)
	}

	usuarioID, err := usuarios.Create(context.Background(), novoUsuario(cargoID, nome, email))
	if err != nil {
		t.Fatalf("não criei o usuário do cenário: %v", err)
	}

	return usuarioID
}

func (c *cenarioBancoSolicitacoes) novaSolicitacao(t *testing.T) *domainsolicitacao.Solicitacao {
	t.Helper()

	valor, err := domainsolicitacao.NovoValor(150000)
	if err != nil {
		t.Fatalf("valor de teste inválido: %v", err)
	}
	prazo, err := domainsolicitacao.NovoPrazoPagamento(time.Now().UTC().AddDate(0, 0, 7))
	if err != nil {
		t.Fatalf("prazo de teste inválido: %v", err)
	}
	observacao, err := domainsolicitacao.NovaObservacao("compra de material")
	if err != nil {
		t.Fatalf("observação de teste inválida: %v", err)
	}
	forma, err := domainsolicitacao.NovaFormaPagamento("pix")
	if err != nil {
		t.Fatalf("forma de pagamento de teste inválida: %v", err)
	}

	solicitacao, err := domainsolicitacao.NovaSolicitacao(c.solicitanteID, valor, prazo, observacao, forma, nil)
	if err != nil {
		t.Fatalf("criação da solicitação de teste falhou: %v", err)
	}

	return solicitacao
}

func (c *cenarioBancoSolicitacoes) criarArquivo(t *testing.T, nome, contentType string) *domainsolicitacao.Arquivo {
	t.Helper()

	arquivo := &domainsolicitacao.Arquivo{
		ID:             uuid.New(),
		ProprietarioID: c.solicitanteID,
		Nome:           nome,
		Chave:          c.solicitanteID.String() + "/" + uuid.NewString() + "/" + nome,
		ContentType:    contentType,
		Tamanho:        2048,
		CriadoEm:       time.Now().UTC(),
	}

	if _, err := c.arquivos.Criar(context.Background(), arquivo); err != nil {
		t.Fatalf("não criei o arquivo de teste: %v", err)
	}

	return arquivo
}

func TestSolicitacaoCriadaPersisteComArquivosEHistorico(t *testing.T) {
	c := cenarioSolicitacoes(t)
	ctx := context.Background()

	arquivo := c.criarArquivo(t, "orcamento.pdf", "application/pdf")
	solicitacao := c.novaSolicitacao(t)
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.solicitanteID, nil, solicitacao.Status, "solicitação criada",
	)

	id, err := c.solicitacoes.Criar(ctx, solicitacao, []uuid.UUID{arquivo.ID}, historico)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	if id != solicitacao.ID {
		t.Errorf("id = %s, esperado %s", id, solicitacao.ID)
	}

	salva, err := c.solicitacoes.Obter(ctx, id)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salva == nil {
		t.Fatal("solicitação criada não encontrada")
	}
	if salva.Valor.Centavos() != 150000 || salva.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("solicitação = %+v, esperado 150000 pendente", salva)
	}
	if salva.Observacao.Texto() != "compra de material" || salva.FormaPagamento != domainsolicitacao.FormaPagamentoPix {
		t.Errorf("textos = %q/%q", salva.Observacao.Texto(), salva.FormaPagamento)
	}
	if salva.Prazo.Data().IsZero() {
		t.Error("prazo voltou zerado")
	}

	arquivos, err := c.solicitacoes.ListarArquivos(ctx, id)
	if err != nil {
		t.Fatalf("arquivos falhou: %v", err)
	}
	if len(arquivos) != 1 || arquivos[0].ID != arquivo.ID {
		t.Errorf("arquivos = %+v, esperado o arquivo criado", arquivos)
	}

	registros, err := c.solicitacoes.ListarHistorico(ctx, id)
	if err != nil {
		t.Fatalf("histórico falhou: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("%d registros, esperado 1", len(registros))
	}
	if registros[0].DeStatus != nil || registros[0].ParaStatus != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("registro = %+v, esperado transição inicial", registros[0])
	}
	if registros[0].Descricao != "solicitação criada" {
		t.Errorf("descrição = %q", registros[0].Descricao)
	}
}

func TestAtualizarStatusSoMudaSeOLStatusAnteriorBater(t *testing.T) {
	c := cenarioSolicitacoes(t)
	ctx := context.Background()

	solicitacao := c.novaSolicitacao(t)
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.solicitanteID, nil, solicitacao.Status, "solicitação criada",
	)
	if _, err := c.solicitacoes.Criar(ctx, solicitacao, nil, historico); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	aprovadorID := c.terceiroID
	agora := time.Now().UTC()
	statusAnterior := solicitacao.Status
	if err := solicitacao.Aprovar(aprovadorID, agora); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	historicoAprovacao := domainsolicitacao.NovoHistorico(
		solicitacao.ID, aprovadorID, &statusAnterior, solicitacao.Status, "solicitação aprovada",
	)

	atualizado, err := c.solicitacoes.AtualizarStatus(ctx, solicitacao, historicoAprovacao)
	if err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}
	if !atualizado {
		t.Fatal("primeira atualização não manteve nenhuma linha")
	}

	atualizado, err = c.solicitacoes.AtualizarStatus(ctx, solicitacao, historicoAprovacao)
	if err != nil {
		t.Fatalf("segunda atualização falhou: %v", err)
	}
	if atualizado {
		t.Error("guarda de status deixou atualizar duas vezes com o mesmo status anterior")
	}

	salva, err := c.solicitacoes.Obter(ctx, solicitacao.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salva.Status != domainsolicitacao.StatusAprovado {
		t.Errorf("status = %q, esperado %q", salva.Status, domainsolicitacao.StatusAprovado)
	}
	if salva.AprovadorID == nil || *salva.AprovadorID != aprovadorID {
		t.Errorf("aprovador = %v, esperado %s", salva.AprovadorID, aprovadorID)
	}
}

func TestCreatePagamentoGravaValorEFicaUnicoPorSolicitacao(t *testing.T) {
	c := cenarioSolicitacoes(t)
	ctx := context.Background()

	solicitacao := c.novaSolicitacao(t)
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.solicitanteID, nil, solicitacao.Status, "solicitação criada",
	)
	if _, err := c.solicitacoes.Criar(ctx, solicitacao, nil, historico); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	statusAnterior := solicitacao.Status
	if err := solicitacao.Aprovar(c.terceiroID, time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	aprovacao := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &statusAnterior, solicitacao.Status, "solicitação aprovada",
	)
	if _, err := c.solicitacoes.AtualizarStatus(ctx, solicitacao, aprovacao); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	arquivo := c.criarArquivo(t, "comprovante.pdf", "application/pdf")
	statusAnterior = solicitacao.Status
	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	valor, err := domainsolicitacao.NovoValor(148500)
	if err != nil {
		t.Fatalf("valor de teste inválido: %v", err)
	}
	pagamento := domainsolicitacao.NovoPagamento(solicitacao.ID, valor, &arquivo.ID, time.Now().UTC())
	pagamentoHistorico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &statusAnterior, solicitacao.Status, "pagamento registrado",
	)

	if err := c.solicitacoes.CriarPagamento(ctx, solicitacao, pagamento, pagamentoHistorico); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	salvo, err := c.solicitacoes.ObterPagamento(ctx, solicitacao.ID)
	if err != nil {
		t.Fatalf("busca do pagamento falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("pagamento não encontrado")
	}
	if salvo.Valor.Centavos() != 148500 {
		t.Errorf("valor = %d, esperado 148500", salvo.Valor.Centavos())
	}
	if salvo.ComprovanteArquivoID == nil || *salvo.ComprovanteArquivoID != arquivo.ID {
		t.Errorf("comprovante = %v, esperado %s", salvo.ComprovanteArquivoID, arquivo.ID)
	}

	solicitacaoPaga, err := c.solicitacoes.Obter(ctx, solicitacao.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if solicitacaoPaga.Status != domainsolicitacao.StatusPago {
		t.Errorf("status = %q, esperado %q", solicitacaoPaga.Status, domainsolicitacao.StatusPago)
	}

	segundaVez := domainsolicitacao.NovoPagamento(solicitacao.ID, valor, nil, time.Now().UTC())
	historicoDuplicado := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &statusAnterior, solicitacao.Status, "pagamento registrado",
	)
	if err := c.solicitacoes.CriarPagamento(ctx, solicitacaoPaga, segundaVez, historicoDuplicado); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segundo pagamento = %v, esperado erro de conflito", err)
	}
}

func TestListarSolicitacoesAplicaFiltros(t *testing.T) {
	c := cenarioSolicitacoes(t)
	ctx := context.Background()

	primeira := c.novaSolicitacao(t)
	segunda := c.novaSolicitacao(t)
	segunda.SolicitanteID = c.terceiroID

	for _, solicitacao := range []*domainsolicitacao.Solicitacao{primeira, segunda} {
		historico := domainsolicitacao.NovoHistorico(
			solicitacao.ID, solicitacao.SolicitanteID, nil, solicitacao.Status, "solicitação criada",
		)
		if _, err := c.solicitacoes.Criar(ctx, solicitacao, nil, historico); err != nil {
			t.Fatalf("criação falhou: %v", err)
		}
	}

	porSolicitante, err := c.solicitacoes.Listar(ctx, portsout.SolicitacaoFiltro{
		PaginacaoFiltro: todasAsPaginas,
		Solicitante:     &c.solicitanteID,
	})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(porSolicitante) != 1 || porSolicitante[0].ID != primeira.ID {
		t.Errorf("filtro por solicitante = %+v, esperado apenas a primeira", porSolicitante)
	}

	statusAprovado := domainsolicitacao.StatusAprovado
	porStatus, err := c.solicitacoes.Listar(ctx, portsout.SolicitacaoFiltro{
		PaginacaoFiltro: todasAsPaginas,
		Status:          statusAprovado,
	})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(porStatus) != 0 {
		t.Errorf("filtro por status aprovado devolveu %d, esperado 0", len(porStatus))
	}

	todas, err := c.solicitacoes.Listar(ctx, portsout.SolicitacaoFiltro{PaginacaoFiltro: todasAsPaginas})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(todas) != 2 {
		t.Errorf("%d solicitações, esperado 2", len(todas))
	}
}

func TestArquivoGuardaVinculoESolicitacaoDono(t *testing.T) {
	c := cenarioSolicitacoes(t)
	ctx := context.Background()

	arquivo := c.criarArquivo(t, "nota.pdf", "application/pdf")

	vinculado, err := c.arquivos.VinculadoASolicitacao(ctx, arquivo.ID)
	if err != nil {
		t.Fatalf("consulta de vínculo falhou: %v", err)
	}
	if vinculado {
		t.Fatal("arquivo saiu vinculado antes de criar a solicitação")
	}

	solicitacao := c.novaSolicitacao(t)
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.solicitanteID, nil, solicitacao.Status, "solicitação criada",
	)
	if _, err := c.solicitacoes.Criar(ctx, solicitacao, []uuid.UUID{arquivo.ID}, historico); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	vinculado, err = c.arquivos.VinculadoASolicitacao(ctx, arquivo.ID)
	if err != nil {
		t.Fatalf("consulta de vínculo falhou: %v", err)
	}
	if !vinculado {
		t.Error("arquivo não ficou vinculado")
	}

	solicitacaoID, err := c.arquivos.SolicitacaoDoArquivo(ctx, arquivo.ID)
	if err != nil {
		t.Fatalf("busca da solicitação do arquivo falhou: %v", err)
	}
	if solicitacaoID == nil || *solicitacaoID != solicitacao.ID {
		t.Errorf("solicitação do arquivo = %v, esperada %s", solicitacaoID, solicitacao.ID)
	}

	porProprietario, err := c.arquivos.ListarPorProprietario(ctx, c.solicitanteID, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(porProprietario) != 1 {
		t.Errorf("%d arquivos, esperado 1", len(porProprietario))
	}

	if err := c.arquivos.Remover(ctx, arquivo.ID); err == nil {
		t.Error("remoção de arquivo vinculado devia falhar pela restrição")
	}
}

func TestAprovadorUnicoPorUsuarioEConsultavelPorId(t *testing.T) {
	c := cenarioSolicitacoes(t)
	ctx := context.Background()

	aprovador := &domainsolicitacao.Aprovador{
		ID:        uuid.New(),
		UsuarioID: c.solicitanteID,
		CriadoEm:  time.Now().UTC(),
	}

	id, err := c.aprovadores.Criar(ctx, aprovador)
	if err != nil {
		t.Fatalf("designação falhou: %v", err)
	}

	if _, err := c.aprovadores.Criar(ctx, &domainsolicitacao.Aprovador{
		ID:        uuid.New(),
		UsuarioID: c.solicitanteID,
		CriadoEm:  time.Now().UTC(),
	}); err == nil {
		t.Error("segunda designação do mesmo usuário devia violar a unicidade")
	}

	porId, err := c.aprovadores.Obter(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if porId == nil || porId.UsuarioID != c.solicitanteID {
		t.Errorf("aprovador = %+v, esperado o usuário do cenário", porId)
	}

	porUsuario, err := c.aprovadores.ObterPorUsuarioID(ctx, c.solicitanteID)
	if err != nil {
		t.Fatalf("busca por usuário falhou: %v", err)
	}
	if porUsuario == nil || porUsuario.ID != id {
		t.Errorf("aprovador = %+v, esperado o registro criado", porUsuario)
	}

	inexistente, err := c.aprovadores.ObterPorUsuarioID(ctx, c.terceiroID)
	if err != nil {
		t.Fatalf("busca por usuário falhou: %v", err)
	}
	if inexistente != nil {
		t.Errorf("usuário sem designação devolveu %+v", inexistente)
	}

	listagem, err := c.aprovadores.Listar(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(listagem) != 1 {
		t.Errorf("%d aprovadores, esperado 1", len(listagem))
	}

	if err := c.aprovadores.Remover(ctx, id); err != nil {
		t.Fatalf("remoção falhou: %v", err)
	}

	aposRemover, err := c.aprovadores.Obter(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if aposRemover != nil {
		t.Errorf("aprovador removido continuou cadastrado: %+v", aposRemover)
	}
}
