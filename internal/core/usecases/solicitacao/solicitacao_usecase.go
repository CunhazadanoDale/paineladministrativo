package solicitacao

import (
	"context"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

var _ portsin.SolicitacaoUseCase = (*SolicitacaoUsecaseImpl)(nil)

type SolicitacaoUsecaseImpl struct {
	repo         portsout.SolicitacaoRepository
	arquivos     portsout.ArquivoRepository
	fornecedores portsout.FornecedorRepository
	categorias   portsout.CategoriaFinanceiraRepository
	contas       portsout.ContaBancariaRepository
	permissoes
}

func NewSolicitacaoUsecase(
	repo portsout.SolicitacaoRepository,
	arquivos portsout.ArquivoRepository,
	aprovadores portsout.AprovadorRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
	fornecedores portsout.FornecedorRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	contas portsout.ContaBancariaRepository,
) *SolicitacaoUsecaseImpl {
	return &SolicitacaoUsecaseImpl{
		repo:         repo,
		arquivos:     arquivos,
		fornecedores: fornecedores,
		categorias:   categorias,
		contas:       contas,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			aprovadores: aprovadores,
			empresas:    empresas,
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

	competencia, err := competenciaInformada(input.Competencia)
	if err != nil {
		return uuid.Nil, err
	}
	classificacao := domainsolicitacao.Classificacao{
		FornecedorID:          input.FornecedorID,
		CategoriaFinanceiraID: input.CategoriaFinanceiraID,
		Competencia:           competencia,
	}

	if err := u.validarEmpresa(ctx, input.SolicitanteID, input.EmpresaID); err != nil {
		return uuid.Nil, err
	}

	if err := validarClassificacao(ctx, u.fornecedores, u.categorias, classificacao); err != nil {
		return uuid.Nil, err
	}

	if err := u.validarArquivos(ctx, input.SolicitanteID, input.ArquivoIDs); err != nil {
		return uuid.Nil, err
	}

	solicitacao, err := domainsolicitacao.NovaSolicitacao(
		input.SolicitanteID, input.EmpresaID, valor, prazo, observacao, formaPagamento,
		classificacao, input.Estimado, input.ArquivoIDs,
	)
	if err != nil {
		return uuid.Nil, err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.SolicitanteID, nil, solicitacao.Status, domainsolicitacao.EventoCriada, "solicitação criada",
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

	acesso, err := u.empresas.AcessoDoUsuario(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if input.Escopo == portsin.EscopoAprovacao || input.Escopo == portsin.EscopoFinanceiro {
		filtro.EmpresaIDs = acesso.EmpresaIDs()
	}
	if input.Empresa != nil {
		if filtro.Solicitante == nil && !acesso.Permite(*input.Empresa) {
			return nil, domain.ErroPermissao("perfil sem acesso a esta empresa")
		}
		filtro.Empresa = input.Empresa
	}

	if err := aplicarFiltrosDeContasAPagar(&filtro, input); err != nil {
		return nil, err
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

func aplicarFiltrosDeContasAPagar(filtro *portsout.SolicitacaoFiltro, input portsin.ListarSolicitacoesInput) error {
	filtro.Fornecedor = input.Fornecedor
	filtro.Categoria = input.Categoria
	filtro.VenceDe = input.VenceDe
	filtro.VenceAte = input.VenceAte

	competencia, err := competenciaInformada(input.Competencia)
	if err != nil {
		return err
	}
	filtro.Competencia = competencia

	if input.Origem != "" {
		origem, err := domainsolicitacao.NovaOrigem(input.Origem)
		if err != nil {
			return err
		}
		filtro.Origem = origem
	}

	if input.VenceDe != nil && input.VenceAte != nil && input.VenceAte.Before(*input.VenceDe) {
		return domain.ErroValidacao("vencimento final não pode ser anterior ao inicial")
	}

	if input.Vencidas {
		hoje := inicioDoDia(time.Now())
		filtro.VencidasEm = &hoje
	}

	return nil
}

func inicioDoDia(data time.Time) time.Time {
	ano, mes, dia := data.UTC().Date()

	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func competenciaInformada(texto string) (*domainsolicitacao.Competencia, error) {
	if strings.TrimSpace(texto) == "" {
		return nil, nil
	}

	competencia, err := domainsolicitacao.NovaCompetencia(texto)
	if err != nil {
		return nil, err
	}

	return &competencia, nil
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
	return u.aprovar(ctx, solicitacaoID, aprovadorID, nil)
}

func (u *SolicitacaoUsecaseImpl) AprovarNaVersao(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID, versao int) error {
	return u.aprovar(ctx, solicitacaoID, aprovadorID, &versao)
}

func (u *SolicitacaoUsecaseImpl) aprovar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID, versao *int) error {
	return u.transicionarPorAprovador(ctx, solicitacaoID, aprovadorID, func(solicitacao *domainsolicitacao.Solicitacao, agora time.Time) (*domainsolicitacao.Historico, error) {
		if versao != nil {
			if err := solicitacao.ConferirVersao(*versao); err != nil {
				return nil, err
			}
		}
		statusAnterior := solicitacao.Status
		if err := solicitacao.Aprovar(aprovadorID, agora); err != nil {
			return nil, err
		}

		return domainsolicitacao.NovoHistorico(
			solicitacao.ID, aprovadorID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoAprovada, "solicitação aprovada",
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
			solicitacao.ID, aprovadorID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoRejeitada,
			"solicitação rejeitada: "+motivoRejeicao.Texto(),
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
		solicitacao.ID, usuarioID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoCancelada, "solicitação cancelada",
	)

	return u.atualizarStatus(ctx, solicitacao, historico)
}

func (u *SolicitacaoUsecaseImpl) Classificar(ctx context.Context, input portsin.ClassificarSolicitacaoInput) error {
	permissao, err := u.ehFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return err
	}
	if !permissao {
		return domain.ErroPermissao("perfil sem permissão para classificar solicitação")
	}

	solicitacao, err := u.buscar(ctx, input.SolicitacaoID)
	if err != nil {
		return err
	}
	if err := u.exigeAcessoAEmpresa(ctx, input.UsuarioID, solicitacao.EmpresaID); err != nil {
		return err
	}
	if err := solicitacao.ConferirVersao(input.Versao); err != nil {
		return err
	}
	if err := solicitacao.ExigirClassificavel(); err != nil {
		return err
	}

	competencia, err := competenciaInformada(input.Competencia)
	if err != nil {
		return err
	}
	classificacao := domainsolicitacao.Classificacao{
		FornecedorID:          input.FornecedorID,
		CategoriaFinanceiraID: input.CategoriaFinanceiraID,
		Competencia:           competencia,
	}

	versaoLida := solicitacao.Versao
	if err := solicitacao.Classificar(classificacao, time.Now().UTC()); err != nil {
		return err
	}
	if err := validarClassificacao(ctx, u.fornecedores, u.categorias, classificacao); err != nil {
		return err
	}

	statusAtual := solicitacao.Status
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.UsuarioID, &statusAtual, statusAtual, domainsolicitacao.EventoClassificada,
		"solicitação classificada na competência "+solicitacao.Competencia.String(),
	)

	return u.repo.Classificar(ctx, solicitacao, versaoLida, historico)
}

func (u *SolicitacaoUsecaseImpl) Editar(ctx context.Context, input portsin.EditarSolicitacaoInput) error {
	solicitacao, err := u.buscar(ctx, input.SolicitacaoID)
	if err != nil {
		return err
	}

	administrador, err := u.ehAdministrador(ctx, input.UsuarioID)
	if err != nil {
		return err
	}
	if !solicitacao.PodeSerEditadaPor(input.UsuarioID, administrador) {
		return domain.ErroPermissao("perfil sem permissão para editar esta solicitação")
	}
	if err := solicitacao.ConferirVersao(input.Versao); err != nil {
		return err
	}
	if err := solicitacao.ExigirEditavel(); err != nil {
		return err
	}

	dados, err := dadosDeEdicao(input, solicitacao)
	if err != nil {
		return err
	}

	versaoLida := solicitacao.Versao
	alterados, err := solicitacao.Editar(dados, time.Now().UTC())
	if err != nil {
		return err
	}
	if len(alterados) == 0 {
		return nil
	}
	if err := validarClassificacao(ctx, u.fornecedores, u.categorias, dados.Classificacao); err != nil {
		return err
	}

	statusAtual := solicitacao.Status
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.UsuarioID, &statusAtual, statusAtual, domainsolicitacao.EventoEditada,
		"solicitação editada: "+strings.Join(alterados, ", "),
	)

	return u.repo.Editar(ctx, solicitacao, versaoLida, historico)
}

func dadosDeEdicao(input portsin.EditarSolicitacaoInput, atual *domainsolicitacao.Solicitacao) (domainsolicitacao.DadosEdicao, error) {
	valor, err := domainsolicitacao.NovoValor(input.ValorCentavos)
	if err != nil {
		return domainsolicitacao.DadosEdicao{}, err
	}

	prazo, err := prazoDaEdicao(input.PrazoPagamento, atual)
	if err != nil {
		return domainsolicitacao.DadosEdicao{}, err
	}

	observacao, err := domainsolicitacao.NovaObservacao(input.Observacao)
	if err != nil {
		return domainsolicitacao.DadosEdicao{}, err
	}

	formaPagamento, err := domainsolicitacao.NovaFormaPagamento(input.FormaPagamento)
	if err != nil {
		return domainsolicitacao.DadosEdicao{}, err
	}

	competencia, err := competenciaInformada(input.Competencia)
	if err != nil {
		return domainsolicitacao.DadosEdicao{}, err
	}

	return domainsolicitacao.DadosEdicao{
		Valor:          valor,
		Prazo:          prazo,
		Observacao:     observacao,
		FormaPagamento: formaPagamento,
		Classificacao: domainsolicitacao.Classificacao{
			FornecedorID:          input.FornecedorID,
			CategoriaFinanceiraID: input.CategoriaFinanceiraID,
			Competencia:           competencia,
		},
		Estimado: input.Estimado,
	}, nil
}

func prazoDaEdicao(informado time.Time, atual *domainsolicitacao.Solicitacao) (domainsolicitacao.PrazoPagamento, error) {
	if !informado.IsZero() && atual.MesmoVencimento(domainsolicitacao.PrazoPagamentoDe(informado)) {
		return atual.Prazo, nil
	}

	return domainsolicitacao.NovoPrazoPagamento(informado)
}

func (u *SolicitacaoUsecaseImpl) Adiar(ctx context.Context, input portsin.AdiarSolicitacaoInput) error {
	permissao, err := u.ehFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return err
	}
	if !permissao {
		return domain.ErroPermissao("perfil sem permissão para adiar solicitação")
	}

	solicitacao, err := u.buscar(ctx, input.SolicitacaoID)
	if err != nil {
		return err
	}
	if err := u.exigeAcessoAEmpresa(ctx, input.UsuarioID, solicitacao.EmpresaID); err != nil {
		return err
	}
	if err := solicitacao.ConferirVersao(input.Versao); err != nil {
		return err
	}
	if err := solicitacao.ExigirAdiavel(); err != nil {
		return err
	}

	vencimento, err := domainsolicitacao.NovoPrazoPagamento(input.Vencimento)
	if err != nil {
		return err
	}
	motivo, err := domainsolicitacao.NovoMotivoAdiamento(input.Motivo)
	if err != nil {
		return err
	}

	versaoLida := solicitacao.Versao
	anterior, err := solicitacao.Adiar(vencimento, time.Now().UTC())
	if err != nil {
		return err
	}

	statusAtual := solicitacao.Status
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.UsuarioID, &statusAtual, statusAtual, domainsolicitacao.EventoAdiada,
		"vencimento adiado de "+anterior.String()+" para "+solicitacao.Prazo.String()+": "+motivo.Texto(),
	)

	return u.repo.Adiar(ctx, solicitacao, versaoLida, historico)
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
	if err := u.exigeAcessoAEmpresa(ctx, input.UsuarioID, solicitacao.EmpresaID); err != nil {
		return err
	}
	if err := solicitacao.ExigirPagavel(); err != nil {
		return err
	}
	if err := validarClassificacao(ctx, u.fornecedores, u.categorias, domainsolicitacao.Classificacao{
		CategoriaFinanceiraID: solicitacao.CategoriaFinanceiraID,
	}); err != nil {
		return err
	}

	if err := u.validarContaDoPagamento(ctx, input.ContaBancariaID, solicitacao.EmpresaID); err != nil {
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
	pagamento, err := domainsolicitacao.NovoPagamento(
		solicitacao.ID, *input.ContaBancariaID, input.UsuarioID, valor, input.ComprovanteArquivoID, input.PagoEm.UTC(), agora,
	)
	if err != nil {
		return err
	}

	statusAnterior := solicitacao.Status
	if err := solicitacao.MarcarComoPago(agora); err != nil {
		return err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.UsuarioID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoPaga, "pagamento registrado",
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
		return nil, domainsolicitacao.ErroPagamentoNaoEncontrado()
	}

	return pagamento, nil
}

func (u *SolicitacaoUsecaseImpl) ListarPagamentos(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Pagamento, error) {
	if _, err := u.buscarVisivel(ctx, solicitacaoID, usuarioID); err != nil {
		return nil, err
	}

	return u.repo.ListarPagamentos(ctx, solicitacaoID)
}

func (u *SolicitacaoUsecaseImpl) EstornarPagamento(ctx context.Context, input portsin.EstornarPagamentoInput) error {
	permissao, err := u.ehFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return err
	}
	if !permissao {
		return domain.ErroPermissao("perfil sem permissão para estornar pagamento")
	}

	solicitacao, err := u.buscar(ctx, input.SolicitacaoID)
	if err != nil {
		return err
	}
	if err := u.exigeAcessoAEmpresa(ctx, input.UsuarioID, solicitacao.EmpresaID); err != nil {
		return err
	}
	if err := solicitacao.ConferirVersao(input.Versao); err != nil {
		return err
	}

	motivo, err := domainsolicitacao.NovoMotivoEstorno(input.Motivo)
	if err != nil {
		return err
	}

	agora := time.Now().UTC()
	versaoLida := solicitacao.Versao
	statusAnterior := solicitacao.Status
	if err := solicitacao.Estornar(agora); err != nil {
		return err
	}

	pagamento, err := u.repo.ObterPagamento(ctx, solicitacao.ID)
	if err != nil {
		return err
	}
	if pagamento == nil {
		return domainsolicitacao.ErroPagamentoNaoEncontrado()
	}
	if _, err := pagamento.Estornar(motivo, input.UsuarioID, input.EstornadoEm.UTC(), agora); err != nil {
		return err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, input.UsuarioID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoEstornada,
		"pagamento estornado: "+motivo.Texto(),
	)

	return u.repo.Estornar(ctx, solicitacao, versaoLida, pagamento, historico)
}

func (u *SolicitacaoUsecaseImpl) validarContaDoPagamento(ctx context.Context, contaBancariaID *uuid.UUID, empresaID uuid.UUID) error {
	if contaBancariaID == nil || *contaBancariaID == uuid.Nil {
		return domainsolicitacao.ErroContaBancariaObrigatoria()
	}

	conta, err := u.contas.Obter(ctx, *contaBancariaID)
	if err != nil {
		return err
	}
	if conta == nil {
		return domainsolicitacao.ErroContaBancariaNaoEncontrada()
	}
	if conta.EmpresaID != empresaID {
		return domainsolicitacao.ErroContaBancariaDeOutraEmpresa()
	}
	if !conta.Ativa {
		return domainsolicitacao.ErroContaBancariaInativa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil || !empresa.Ativa {
		return domainsolicitacao.ErroEmpresaInativaNoPagamento()
	}

	return nil
}

func (u *SolicitacaoUsecaseImpl) Pendencias(ctx context.Context, usuarioID uuid.UUID) (domainsolicitacao.Pendencias, error) {
	acesso, err := u.empresas.AcessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return domainsolicitacao.Pendencias{}, err
	}
	empresaIDs := acesso.EmpresaIDs()

	var pendencias domainsolicitacao.Pendencias

	aprovador, err := u.ehAprovador(ctx, usuarioID)
	if err != nil {
		return domainsolicitacao.Pendencias{}, err
	}
	if aprovador {
		total, err := u.repo.Contar(ctx, portsout.SolicitacaoFiltro{Status: domainsolicitacao.StatusPendenteAprovacao, EmpresaIDs: empresaIDs})
		if err != nil {
			return domainsolicitacao.Pendencias{}, err
		}
		pendencias.Aprovacao = &total
	}

	financeiro, err := u.ehFinanceiro(ctx, usuarioID)
	if err != nil {
		return domainsolicitacao.Pendencias{}, err
	}
	if !financeiro {
		return pendencias, nil
	}

	aPagar, err := u.repo.Contar(ctx, portsout.SolicitacaoFiltro{Status: domainsolicitacao.StatusAprovado, EmpresaIDs: empresaIDs})
	if err != nil {
		return domainsolicitacao.Pendencias{}, err
	}
	hoje := inicioDoDia(time.Now())
	vencidas, err := u.repo.Contar(ctx, portsout.SolicitacaoFiltro{VencidasEm: &hoje, EmpresaIDs: empresaIDs})
	if err != nil {
		return domainsolicitacao.Pendencias{}, err
	}
	pendencias.Financeiro = &aPagar
	pendencias.Vencidas = &vencidas

	return pendencias, nil
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
	if err := u.exigeAcessoAEmpresa(ctx, aprovadorID, solicitacao.EmpresaID); err != nil {
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

func (u *SolicitacaoUsecaseImpl) validarEmpresa(ctx context.Context, solicitanteID, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa da solicitação não informada")
	}

	if err := u.exigeAcessoAEmpresa(ctx, solicitanteID, empresaID); err != nil {
		return err
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domainsolicitacao.ErroEmpresaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainsolicitacao.ErroEmpresaInativa()
	}

	return nil
}

func validarClassificacao(
	ctx context.Context,
	fornecedores portsout.FornecedorRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	classificacao domainsolicitacao.Classificacao,
) error {
	if classificacao.FornecedorID != nil {
		fornecedor, err := fornecedores.Obter(ctx, *classificacao.FornecedorID)
		if err != nil {
			return err
		}
		if fornecedor == nil {
			return domainsolicitacao.ErroFornecedorNaoEncontrado()
		}
		if !fornecedor.Ativo {
			return domainsolicitacao.ErroFornecedorInativo()
		}
	}

	if classificacao.CategoriaFinanceiraID == nil {
		return nil
	}

	categoria, err := categorias.Obter(ctx, *classificacao.CategoriaFinanceiraID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domainsolicitacao.ErroCategoriaFinanceiraNaoEncontrada()
	}
	if !categoria.Lancavel() {
		return domainsolicitacao.ErroCategoriaFinanceiraNaoLancavel()
	}
	if categoria.Natureza != domainfinanceiro.NaturezaDespesa {
		return domainsolicitacao.ErroCategoriaFinanceiraNaoEhDespesa()
	}

	return nil
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
		if arquivo.ContentType == contentTypeXML {
			return domain.ErroValidacao("arquivo XML é exclusivo da importação de nota fiscal e não pode ser anexado à solicitação")
		}
		if arquivo.ContentType == contentTypeOFX {
			return domain.ErroValidacao("arquivo OFX é exclusivo da importação de extrato e não pode ser anexado à solicitação")
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
