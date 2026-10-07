package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func TestCriarSolicitacaoValidaFicaPendenteComHistorico(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	arquivo := c.enviarArquivo(t, solicitante, "orcamento.pdf", "application/pdf")

	id, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
		SolicitanteID:  solicitante,
		ValorCentavos:  150000,
		PrazoPagamento: time.Now().UTC().AddDate(0, 0, 10),
		Observacao:     "compra de cimento",
		FormaPagamento: "boleto",
		ArquivoIDs:     []uuid.UUID{arquivo.ID},
	})
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, solicitante)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("status = %q, esperado %q", salva.Status, domainsolicitacao.StatusPendenteAprovacao)
	}
	if salva.Valor.Centavos() != 150000 {
		t.Errorf("valor = %d, esperado 150000", salva.Valor.Centavos())
	}

	arquivos, err := c.solicitacao.ListarArquivos(c.ctx, id, solicitante)
	if err != nil {
		t.Fatalf("listagem de arquivos falhou: %v", err)
	}
	if len(arquivos) != 1 || arquivos[0].ID != arquivo.ID {
		t.Errorf("arquivos = %+v, esperado o arquivo enviado", arquivos)
	}

	historico, err := c.solicitacao.ListarHistorico(c.ctx, id, solicitante)
	if err != nil {
		t.Fatalf("histórico falhou: %v", err)
	}
	if len(historico) != 1 {
		t.Fatalf("%d registros de histórico, esperado 1", len(historico))
	}
	if historico[0].ParaStatus != domainsolicitacao.StatusPendenteAprovacao || historico[0].DeStatus != nil {
		t.Errorf("primeiro registro = %+v, esperado transição inicial", historico[0])
	}
}

func TestCriarSolicitacaoRejeitaDadosInvalidos(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)

	casos := []string{"valor zerado", "forma inválida", "observação vazia"}

	for _, caso := range casos {
		input := portsin.CriarSolicitacaoInput{
			SolicitanteID:  solicitante,
			ValorCentavos:  10000,
			PrazoPagamento: time.Now().UTC().AddDate(0, 0, 1),
			Observacao:     "observação",
			FormaPagamento: "pix",
		}

		switch caso {
		case "valor zerado":
			input.ValorCentavos = 0
		case "forma inválida":
			input.FormaPagamento = "dinheiro"
		case "observação vazia":
			input.Observacao = "   "
		}

		if _, err := c.solicitacao.Criar(c.ctx, input); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s = %v, esperado erro de validação", caso, err)
		}
	}
}

func TestCriarSolicitacaoComArquivoDeOutroUsuarioRecebeProibido(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	bruno := c.novoUsuario("Bruno", false, false)
	arquivoDoBruno := c.enviarArquivo(t, bruno, "nota.pdf", "application/pdf")

	_, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
		SolicitanteID:  ana,
		ValorCentavos:  10000,
		PrazoPagamento: time.Now().UTC().AddDate(0, 0, 1),
		Observacao:     "observação",
		FormaPagamento: "pix",
		ArquivoIDs:     []uuid.UUID{arquivoDoBruno.ID},
	})
	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("arquivo de terceiro = %v, esperado erro de permissão", err)
	}
}

func TestCriarSolicitacaoComArquivoJaVinculadoRecebeErroDeValidacao(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	arquivo := c.enviarArquivo(t, ana, "nota.pdf", "application/pdf")

	c.criarSolicitacaoComArquivos(t, ana, []uuid.UUID{arquivo.ID})

	_, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
		SolicitanteID:  ana,
		ValorCentavos:  10000,
		PrazoPagamento: time.Now().UTC().AddDate(0, 0, 1),
		Observacao:     "observação",
		FormaPagamento: "pix",
		ArquivoIDs:     []uuid.UUID{arquivo.ID},
	})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("arquivo reutilizado = %v, esperado erro de validação", err)
	}
}

func TestListarPorEscopoMinhasMostraSoAsDoSolicitante(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	bruno := c.novoUsuario("Bruno", false, false)
	c.criarSolicitacao(t, ana, 10000)
	c.criarSolicitacao(t, bruno, 20000)

	itens, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: ana,
		Escopo:    portsin.EscopoMinhas,
	})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(itens) != 1 {
		t.Fatalf("%d solicitações, esperado 1", len(itens))
	}
	if itens[0].SolicitanteID != ana {
		t.Errorf("solicitante = %s, esperado %s", itens[0].SolicitanteID, ana)
	}
}

func TestListarPorEscopoAprovacaoExigePerfil(t *testing.T) {
	c := novoCenario(t)
	comum := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)

	if _, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: comum,
		Escopo:    portsin.EscopoAprovacao,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("escopo aprovação sem perfil = %v, esperado erro de permissão", err)
	}

	c.designar(t, aprovador)
	c.criarSolicitacao(t, comum, 10000)

	itens, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: aprovador,
		Escopo:    portsin.EscopoAprovacao,
	})
	if err != nil {
		t.Fatalf("listagem do aprovador falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("fila de aprovação = %+v, esperado apenas pendentes", itens)
	}
}

func TestListarPorEscopoFinanceiroExigePerfil(t *testing.T) {
	c := novoCenario(t)
	comum := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Bruno", false, true)
	aprovador := c.novoUsuario("Carla", false, false)
	c.designar(t, aprovador)

	pendente := c.criarSolicitacao(t, comum, 10000)
	aprovada := c.criarSolicitacao(t, comum, 20000)

	if err := c.solicitacao.Aprovar(c.ctx, aprovada, aprovador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	if _, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: comum,
		Escopo:    portsin.EscopoFinanceiro,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("escopo financeiro sem perfil = %v, esperado erro de permissão", err)
	}

	itens, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: financeiro,
		Escopo:    portsin.EscopoFinanceiro,
	})
	if err != nil {
		t.Fatalf("listagem do financeiro falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].ID != aprovada {
		t.Errorf("fila do financeiro = %+v, esperado apenas a %s", itens, aprovada)
	}
	if pendente == aprovada {
		t.Fatal("ids de teste iguais")
	}
}

func TestListarPorEscopoTodasExigeAdministracao(t *testing.T) {
	c := novoCenario(t)
	comum := c.novoUsuario("Ana", false, false)
	admin := c.novoUsuario("Bruno", true, false)
	c.criarSolicitacao(t, comum, 10000)
	c.criarSolicitacao(t, comum, 20000)

	if _, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: comum,
		Escopo:    portsin.EscopoTodas,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("escopo todas sem perfil = %v, esperado erro de permissão", err)
	}

	itens, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: admin,
		Escopo:    portsin.EscopoTodas,
	})
	if err != nil {
		t.Fatalf("listagem do administrador falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Errorf("%d solicitações, esperado 2", len(itens))
	}
}

func TestListarComEscopoDesconhecidoRecebeErroDeValidacao(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)

	if _, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: ana,
		Escopo:    "outra-coisa",
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("escopo desconhecido = %v, esperado erro de validação", err)
	}
}

func TestAprovarSemSerAprovadorRecebeProibido(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	bruno := c.novoUsuario("Bruno", false, false)
	id := c.criarSolicitacao(t, ana, 10000)

	if err := c.solicitacao.Aprovar(c.ctx, id, bruno); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("aprovação sem perfil = %v, esperado erro de permissão", err)
	}
}

func TestAprovarComDesignacaoMudaStatusERegistraHistorico(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	id := c.criarSolicitacao(t, ana, 10000)

	if err := c.solicitacao.Aprovar(c.ctx, id, aprovador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, ana)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Status != domainsolicitacao.StatusAprovado {
		t.Errorf("status = %q, esperado %q", salva.Status, domainsolicitacao.StatusAprovado)
	}
	if salva.AprovadorID == nil || *salva.AprovadorID != aprovador {
		t.Errorf("aprovador = %v, esperado %s", salva.AprovadorID, aprovador)
	}

	historico, err := c.solicitacao.ListarHistorico(c.ctx, id, ana)
	if err != nil {
		t.Fatalf("histórico falhou: %v", err)
	}
	if len(historico) != 2 {
		t.Fatalf("%d registros, esperado 2", len(historico))
	}
	if *historico[0].DeStatus != domainsolicitacao.StatusPendenteAprovacao ||
		historico[0].ParaStatus != domainsolicitacao.StatusAprovado {
		t.Errorf("registro = %+v, esperado pendente -> aprovado", historico[0])
	}
}

func TestAprovarSolicitacaoJaAprovadaRecebeConflito(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	id := c.criarSolicitacao(t, ana, 10000)

	if err := c.solicitacao.Aprovar(c.ctx, id, aprovador); err != nil {
		t.Fatalf("primeira aprovação falhou: %v", err)
	}
	if err := c.solicitacao.Aprovar(c.ctx, id, aprovador); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segunda aprovação = %v, esperado erro de conflito", err)
	}
}

func TestRejeitarExigeMotivoValido(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	id := c.criarSolicitacao(t, ana, 10000)

	if err := c.solicitacao.Rejeitar(c.ctx, id, aprovador, "   "); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("motivo vazio = %v, esperado erro de validação", err)
	}

	if err := c.solicitacao.Rejeitar(c.ctx, id, aprovador, "fora do orçamento"); err != nil {
		t.Fatalf("rejeição falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, ana)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Status != domainsolicitacao.StatusRejeitado {
		t.Errorf("status = %q, esperado %q", salva.Status, domainsolicitacao.StatusRejeitado)
	}
	if salva.MotivoRejeicao == nil || salva.MotivoRejeicao.Texto() != "fora do orçamento" {
		t.Errorf("motivo = %v, esperado registrado", salva.MotivoRejeicao)
	}
}

func TestCancelarSomentePeloSolicitanteOuAdministrador(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	estranho := c.novoUsuario("Bruno", false, false)
	admin := c.novoUsuario("Carla", true, false)
	id := c.criarSolicitacao(t, ana, 10000)

	if err := c.solicitacao.Cancelar(c.ctx, id, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Fatalf("cancelamento por terceiro = %v, esperado erro de permissão", err)
	}

	if err := c.solicitacao.Cancelar(c.ctx, id, admin); err != nil {
		t.Fatalf("cancelamento pelo administrador falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, ana)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Status != domainsolicitacao.StatusCancelado {
		t.Errorf("status = %q, esperado %q", salva.Status, domainsolicitacao.StatusCancelado)
	}

	if err := c.solicitacao.Cancelar(c.ctx, id, ana); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("recancelamento = %v, esperado erro de conflito", err)
	}
}

func TestRegistrarPagamentoExigePerfilFinanceiro(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	id := c.criarSolicitacao(t, ana, 10000)

	if err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
		SolicitacaoID: id,
		UsuarioID:     ana,
		ValorCentavos: 10000,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("pagamento sem perfil = %v, esperado erro de permissão", err)
	}
}

func TestRegistrarPagamentoAceitaValorDiferenteDoEstimado(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Bruno", false, true)
	aprovador := c.novoUsuario("Carla", false, false)
	c.designar(t, aprovador)
	id := c.criarSolicitacao(t, ana, 150000)

	if err := c.solicitacao.Aprovar(c.ctx, id, aprovador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	comprovante := c.enviarArquivo(t, financeiro, "comprovante.pdf", "application/pdf")

	if err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
		SolicitacaoID:        id,
		UsuarioID:            financeiro,
		ValorCentavos:        148500,
		ComprovanteArquivoID: &comprovante.ID,
		PagoEm:               time.Now().UTC(),
	}); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, ana)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Status != domainsolicitacao.StatusPago {
		t.Errorf("status = %q, esperado %q", salva.Status, domainsolicitacao.StatusPago)
	}
	if salva.Valor.Centavos() != 150000 {
		t.Errorf("valor estimado alterado para %d", salva.Valor.Centavos())
	}

	pagamento, err := c.solicitacao.ObterPagamento(c.ctx, id, ana)
	if err != nil {
		t.Fatalf("consulta do pagamento falhou: %v", err)
	}
	if pagamento.Valor.Centavos() != 148500 {
		t.Errorf("valor pago = %d, esperado 148500", pagamento.Valor.Centavos())
	}
	if pagamento.ComprovanteArquivoID == nil || *pagamento.ComprovanteArquivoID != comprovante.ID {
		t.Errorf("comprovante = %v, esperado %s", pagamento.ComprovanteArquivoID, comprovante.ID)
	}

	if err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
		SolicitacaoID: id,
		UsuarioID:     financeiro,
		ValorCentavos: 100,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("pagamento repetido = %v, esperado erro de conflito", err)
	}
}

func TestConsultasDaSolicitacaoExigemVisibilidade(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	estranho := c.novoUsuario("Bruno", false, false)
	id := c.criarSolicitacao(t, ana, 10000)

	if _, err := c.solicitacao.Obter(c.ctx, id, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("consulta por terceiro = %v, esperado erro de permissão", err)
	}
	if _, err := c.solicitacao.ListarHistorico(c.ctx, id, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("histórico por terceiro = %v, esperado erro de permissão", err)
	}
	if _, err := c.solicitacao.ListarArquivos(c.ctx, id, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("arquivos por terceiro = %v, esperado erro de permissão", err)
	}
	if _, err := c.solicitacao.ObterPagamento(c.ctx, id, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("pagamento por terceiro = %v, esperado erro de permissão", err)
	}
}

func TestObterPagamentoSemRegistroDevolveNaoEncontrado(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	id := c.criarSolicitacao(t, ana, 10000)

	if _, err := c.solicitacao.ObterPagamento(c.ctx, id, ana); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("consulta sem pagamento = %v, esperado não encontrado", err)
	}
}

func TestFinanceiroSoVeSolicitacoesAprovadas(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Bruno", false, true)
	aprovador := c.novoUsuario("Carla", false, false)
	c.designar(t, aprovador)
	pendente := c.criarSolicitacao(t, ana, 10000)

	if _, err := c.solicitacao.Obter(c.ctx, pendente, financeiro); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("financeiro em solicitação pendente = %v, esperado erro de permissão", err)
	}

	aprovada := c.criarSolicitacao(t, ana, 20000)
	if err := c.solicitacao.Aprovar(c.ctx, aprovada, aprovador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	if _, err := c.solicitacao.Obter(c.ctx, aprovada, financeiro); err != nil {
		t.Errorf("financeiro em solicitação aprovada = %v, esperado permitido", err)
	}
}

func TestObterSolicitacaoInexistenteRetornaNaoEncontrado(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)

	if _, err := c.solicitacao.Obter(c.ctx, uuid.New(), ana); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("consulta inexistente = %v, esperado não encontrado", err)
	}
}
