package solicitacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var _ portsin.SolicitacaoUseCase = (*SolicitacaoUsecaseImpl)(nil)

type SolicitacaoUsecaseImpl struct {
	repo     portsout.SolicitacaoRepository
	arquivos portsout.ArquivoRepository
	permissoes
}

func NewSolicitacaoUsecase(
	repo portsout.SolicitacaoRepository,
	arquivos portsout.ArquivoRepository,
	aprovadores portsout.AprovadorRepository,
	usuarios portsoutusuarios.UsuarioRepository,
	cargos portsoutusuarios.CargoRepository,
) *SolicitacaoUsecaseImpl {
	return &SolicitacaoUsecaseImpl{
		repo:     repo,
		arquivos: arquivos,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			aprovadores: aprovadores,
		},
	}
}

func (u *SolicitacaoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarSolicitacaoInput) (uuid.UUID, error) {
	valor, err := domainsolicitacao.NovoValor(input.ValorCentavos)
	if err != nil {
		return uuid.Nil, err
	}

	prazo, err := domainsolicitacao.NovoPrazoPagamento(input.PrazoPagamento)
	if err != nil {
		return uuid.Nil, err
	}

	observacao, err := domainsolicitacao.NovaObservacao(input.Observacao)
	if err != nil {
		return uuid.Nil, err
	}

	formaPagamento, err := domainsolicitacao.NovaFormaPagamento(input.FormaPagamento)
	if err != nil {
		return uuid.Nil, err
	}

	if err := u.validarArquivos(ctx, input.SolicitanteID, input.ArquivoIDs); err != nil {
		return uuid.Nil, err
	}

	solicitacao, err := domainsolicitacao.NovaSolicitacao(
		input.SolicitanteID, valor, prazo, observacao, formaPagamento, input.ArquivoIDs,
	)
	if err != nil {
		return uuid.Nil, err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.SolicitanteID, nil, solicitacao.Status, "solicitação criada",
	)

	return u.repo.Criar(ctx, solicitacao, input.ArquivoIDs, historico)
}

func (u *SolicitacaoUsecaseImpl) Obter(ctx context.Context, id, usuarioID uuid.UUID) (*domainsolicitacao.Solicitacao, error) {
	return u.buscarVisivel(ctx, id, usuarioID)
}

func (u *SolicitacaoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarSolicitacoesInput) ([]*domainsolicitacao.Solicitacao, error) {
	filtro := portsout.SolicitacaoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
	}

	switch input.Escopo {
	case "", portsin.EscopoMinhas:
		filtro.Solicitante = &input.UsuarioID
	case portsin.EscopoAprovacao:
		permissao, err := u.ehAprovador(ctx, input.UsuarioID)
		if err != nil {
			return nil, err
		}
		if !permissao {
			return nil, domain.ErroPermissao("perfil sem permissão para consultar a fila de aprovação")
		}
		filtro.Status = domainsolicitacao.StatusPendenteAprovacao
	case portsin.EscopoFinanceiro:
		permissao, err := u.ehFinanceiro(ctx, input.UsuarioID)
		if err != nil {
			return nil, err
		}
		if !permissao {
			return nil, domain.ErroPermissao("perfil sem permissão para consultar a fila do financeiro")
		}
		filtro.Status = domainsolicitacao.StatusAprovado
	case portsin.EscopoTodas:
		administrador, err := u.ehAdministrador(ctx, input.UsuarioID)
		if err != nil {
			return nil, err
		}
		if !administrador {
			return nil, domain.ErroPermissao("perfil sem permissão para listar todas as solicitações")
		}
	default:
		return nil, domain.ErroValidacao("escopo de listagem inválido")
	}

	if input.Status != "" {
		status, err := domainsolicitacao.NovoStatus(input.Status)
		if err != nil {
			return nil, err
		}
		if !statusPermitidoNoEscopo(input.Escopo, status) {
			return nil, domain.ErroValidacao("status não permitido para o escopo de listagem informado")
		}
		filtro.Status = status
	}

	return u.repo.Listar(ctx, filtro)
}

func statusPermitidoNoEscopo(escopo string, status domainsolicitacao.Status) bool {
	switch escopo {
	case portsin.EscopoAprovacao:
		return status == domainsolicitacao.StatusPendenteAprovacao
	case portsin.EscopoFinanceiro:
		return status != domainsolicitacao.StatusPendenteAprovacao
	default:
		return true
	}
}

func (u *SolicitacaoUsecaseImpl) Aprovar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID) error {
	return u.transicionarPorAprovador(ctx, solicitacaoID, aprovadorID, func(solicitacao *domainsolicitacao.Solicitacao, agora time.Time) (*domainsolicitacao.Historico, error) {
		statusAnterior := solicitacao.Status
		if err := solicitacao.Aprovar(aprovadorID, agora); err != nil {
			return nil, err
		}

		return domainsolicitacao.NovoHistorico(
			solicitacao.ID, aprovadorID, &statusAnterior, solicitacao.Status, "solicitação aprovada",
		), nil
	})
}

func (u *SolicitacaoUsecaseImpl) Rejeitar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID, motivo string) error {
	motivoRejeicao, err := domainsolicitacao.NovoMotivoRejeicao(motivo)
	if err != nil {
		return err
	}

	return u.transicionarPorAprovador(ctx, solicitacaoID, aprovadorID, func(solicitacao *domainsolicitacao.Solicitacao, agora time.Time) (*domainsolicitacao.Historico, error) {
		statusAnterior := solicitacao.Status
		if err := solicitacao.Rejeitar(aprovadorID, motivoRejeicao, agora); err != nil {
			return nil, err
		}

		return domainsolicitacao.NovoHistorico(
			solicitacao.ID, aprovadorID, &statusAnterior, solicitacao.Status, "solicitação rejeitada: "+motivoRejeicao.Texto(),
		), nil
	})
}

func (u *SolicitacaoUsecaseImpl) Cancelar(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) error {
	solicitacao, err := u.buscar(ctx, solicitacaoID)
	if err != nil {
		return err
	}

	administrador, err := u.ehAdministrador(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !solicitacao.PodeSerCanceladoPor(usuarioID, administrador) {
		return domain.ErroPermissao("perfil sem permissão para cancelar esta solicitação")
	}

	agora := time.Now().UTC()
	statusAnterior := solicitacao.Status
	if err := solicitacao.Cancelar(agora); err != nil {
		return err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, usuarioID, &statusAnterior, solicitacao.Status, "solicitação cancelada",
	)

	return u.atualizarStatus(ctx, solicitacao, historico)
}

func (u *SolicitacaoUsecaseImpl) RegistrarPagamento(ctx context.Context, input portsin.RegistrarPagamentoInput) error {
	permissao, err := u.ehFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return err
	}
	if !permissao {
		return domain.ErroPermissao("perfil sem permissão para registrar pagamento")
	}

	solicitacao, err := u.buscar(ctx, input.SolicitacaoID)
	if err != nil {
		return err
	}

	if err := u.validarComprovante(ctx, solicitacao.ID, input.UsuarioID, input.ComprovanteArquivoID); err != nil {
		return err
	}

	valor, err := domainsolicitacao.NovoValor(input.ValorCentavos)
	if err != nil {
		return err
	}

	agora := time.Now().UTC()
	pagamento, err := domainsolicitacao.NovoPagamento(solicitacao.ID, valor, input.ComprovanteArquivoID, input.PagoEm.UTC(), agora)
	if err != nil {
		return err
	}

	statusAnterior := solicitacao.Status
	if err := solicitacao.MarcarComoPago(agora); err != nil {
		return err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.UsuarioID, &statusAnterior, solicitacao.Status, "pagamento registrado",
	)

	return u.repo.CriarPagamento(ctx, solicitacao, pagamento, historico)
}

func (u *SolicitacaoUsecaseImpl) ListarHistorico(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Historico, error) {
	if _, err := u.buscarVisivel(ctx, solicitacaoID, usuarioID); err != nil {
		return nil, err
	}

	return u.repo.ListarHistorico(ctx, solicitacaoID)
}

func (u *SolicitacaoUsecaseImpl) ListarArquivos(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Arquivo, error) {
	if _, err := u.buscarVisivel(ctx, solicitacaoID, usuarioID); err != nil {
		return nil, err
	}

	return u.repo.ListarArquivos(ctx, solicitacaoID)
}

func (u *SolicitacaoUsecaseImpl) ObterPagamento(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) (*domainsolicitacao.Pagamento, error) {
	if _, err := u.buscarVisivel(ctx, solicitacaoID, usuarioID); err != nil {
		return nil, err
	}

	pagamento, err := u.repo.ObterPagamento(ctx, solicitacaoID)
	if err != nil {
		return nil, err
	}
	if pagamento == nil {
		return nil, domain.ErroNaoEncontrado("pagamento não encontrado")
	}

	return pagamento, nil
}

func (u *SolicitacaoUsecaseImpl) transicionarPorAprovador(
	ctx context.Context,
	solicitacaoID, aprovadorID uuid.UUID,
	aplicar func(*domainsolicitacao.Solicitacao, time.Time) (*domainsolicitacao.Historico, error),
) error {
	permissao, err := u.ehAprovador(ctx, aprovadorID)
	if err != nil {
		return err
	}
	if !permissao {
		return domain.ErroPermissao("perfil sem permissão para aprovar ou rejeitar solicitações")
	}

	solicitacao, err := u.buscar(ctx, solicitacaoID)
	if err != nil {
		return err
	}

	historico, err := aplicar(solicitacao, time.Now().UTC())
	if err != nil {
		return err
	}

	return u.atualizarStatus(ctx, solicitacao, historico)
}

func (u *SolicitacaoUsecaseImpl) atualizarStatus(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) error {
	atualizado, err := u.repo.AtualizarStatus(ctx, solicitacao, historico)
	if err != nil {
		return err
	}
	if !atualizado {
		return domain.ErroConflito("solicitação alterada por outra operação, recarregue e tente novamente")
	}

	return nil
}

func (u *SolicitacaoUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Solicitacao, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id da solicitação não informado")
	}

	solicitacao, err := u.repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if solicitacao == nil {
		return nil, domain.ErrNotFound
	}

	return solicitacao, nil
}

func (u *SolicitacaoUsecaseImpl) buscarVisivel(ctx context.Context, id, usuarioID uuid.UUID) (*domainsolicitacao.Solicitacao, error) {
	solicitacao, err := u.buscar(ctx, id)
	if err != nil {
		return nil, err
	}

	podeVer, err := podeVerSolicitacao(ctx, &u.permissoes, solicitacao, usuarioID)
	if err != nil {
		return nil, err
	}
	if !podeVer {
		return nil, domain.ErroPermissao("perfil sem permissão para consultar esta solicitação")
	}

	return solicitacao, nil
}

func (u *SolicitacaoUsecaseImpl) validarArquivos(ctx context.Context, proprietarioID uuid.UUID, arquivoIDs []uuid.UUID) error {
	vistos := make(map[uuid.UUID]struct{}, len(arquivoIDs))
	for _, arquivoID := range arquivoIDs {
		if arquivoID == uuid.Nil {
			return domain.ErroValidacao("arquivo inválido vinculado à solicitação")
		}
		if _, repetido := vistos[arquivoID]; repetido {
			return domain.ErroValidacao("arquivo repetido na solicitação")
		}
		vistos[arquivoID] = struct{}{}

		arquivo, err := u.arquivos.Obter(ctx, arquivoID)
		if err != nil {
			return err
		}
		if arquivo == nil {
			return domain.ErroValidacao("arquivo vinculado à solicitação não encontrado")
		}
		if arquivo.ProprietarioID != proprietarioID {
			return domain.ErroPermissao("arquivo não pertence ao solicitante")
		}

		vinculado, err := u.arquivos.VinculadoASolicitacao(ctx, arquivoID)
		if err != nil {
			return err
		}
		if vinculado {
			return domain.ErroValidacao("arquivo já vinculado a outra solicitação")
		}
	}

	return nil
}

func (u *SolicitacaoUsecaseImpl) validarComprovante(ctx context.Context, solicitacaoID, usuarioID uuid.UUID, comprovanteID *uuid.UUID) error {
	if comprovanteID == nil {
		return nil
	}

	arquivo, err := u.arquivos.Obter(ctx, *comprovanteID)
	if err != nil {
		return err
	}
	if arquivo == nil {
		return domain.ErroValidacao("comprovante não encontrado")
	}
	if arquivo.ProprietarioID != usuarioID {
		return domain.ErroPermissao("comprovante não pertence ao usuário")
	}
	if arquivo.ContentType != "application/pdf" {
		return domain.ErroValidacao("comprovante deve ser um arquivo PDF")
	}
	vinculado, err := u.arquivos.SolicitacaoDoArquivo(ctx, *comprovanteID)
	if err != nil {
		return err
	}
	if vinculado != nil && *vinculado != solicitacaoID {
		return domain.ErroValidacao("comprovante já vinculado a outra solicitação")
	}

	return nil
}
