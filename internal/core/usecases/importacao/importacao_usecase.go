package importacao

import (
	"context"
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

var _ portsin.ImportacaoUseCase = (*ImportacaoUsecaseImpl)(nil)

type ImportacaoUsecaseImpl struct {
	fornecedores portsout.FornecedorRepository
	produtos     portsout.ProdutoRepository
	permissoes
}

func NewImportacaoUsecase(
	importacoes portsout.ImportacaoRepository,
	fornecedores portsout.FornecedorRepository,
	produtos portsout.ProdutoRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ImportacaoUsecaseImpl {
	return &ImportacaoUsecaseImpl{
		fornecedores: fornecedores,
		produtos:     produtos,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			empresas:    empresas,
			importacoes: importacoes,
		},
	}
}

func (u *ImportacaoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarImportacaoInput) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigirEmpresaAtiva(ctx, perfil, input.EmpresaID); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, nil, input.Dados)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	importacao, err := domainimportacao.NovaImportacao(input.EmpresaID, input.UsuarioID, dados, agora)
	if err != nil {
		return nil, err
	}

	if err := u.importacoes.Gravar(ctx, &portsout.Gravacao{
		Importacao:        importacao,
		Novo:              true,
		FornecedorTrocado: true,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, input.UsuarioID, domainimportacao.AcaoCriada, nil, importacao.Status, "", agora),
		},
	}); err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: true}, nil
}

func (u *ImportacaoUsecaseImpl) Obter(ctx context.Context, importacaoID, usuarioID uuid.UUID) (*portsin.Visao, error) {
	perfil, err := u.exigeLeitura(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, importacaoID)
	if err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: perfil.financeiro}, nil
}

func (u *ImportacaoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarImportacoesInput) (*portsin.Lista, error) {
	perfil, err := u.exigeLeitura(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	filtro := portsout.ImportacaoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		FornecedorID:    input.FornecedorID,
		Numero:          input.Numero,
	}
	if input.Status != "" {
		if filtro.Status, err = domainimportacao.NovoStatus(input.Status); err != nil {
			return nil, err
		}
	}
	if input.ChegadaAte != nil {
		data := domainimportacao.DataDe(*input.ChegadaAte)
		filtro.ChegadaAte = &data
	}

	importacoes, err := u.importacoes.Listar(ctx, filtro)
	if err != nil {
		return nil, err
	}

	return &portsin.Lista{Importacoes: importacoes, ComValores: perfil.financeiro}, nil
}

func (u *ImportacaoUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarImportacaoInput) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, input.ImportacaoID)
	if err != nil {
		return nil, err
	}
	if err := importacao.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, importacao, input.Dados)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	anterior := importacao.Status
	fornecedorTrocado := dados.FornecedorID != importacao.FornecedorID
	if err := importacao.Alterar(dados, agora); err != nil {
		return nil, err
	}

	if err := u.importacoes.Gravar(ctx, &portsout.Gravacao{
		Importacao:        importacao,
		VersaoLida:        input.Versao,
		StatusAnterior:    anterior,
		SubstituirItens:   true,
		FornecedorTrocado: fornecedorTrocado,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, input.UsuarioID, domainimportacao.AcaoAlterada, nil, importacao.Status, "", agora),
		},
	}); err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: true}, nil
}

func (u *ImportacaoUsecaseImpl) Embarcar(ctx context.Context, input portsin.EmbarcarInput) (*portsin.Visao, error) {
	return u.transicionar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao, domainimportacao.AcaoEmbarcada,
		func(importacao *domainimportacao.Importacao, agora time.Time) (string, error) {
			err := importacao.Embarcar(domainimportacao.EmbarqueInput{
				EmbarcadaEm:       input.EmbarcadaEm,
				ChegadaPrevista:   input.ChegadaPrevista,
				DocumentoEmbarque: input.DocumentoEmbarque,
			}, agora)
			if err != nil {
				return "", err
			}

			return "chegada prevista em " + importacao.ChegadaPrevista.Format("02/01/2006"), nil
		})
}

func (u *ImportacaoUsecaseImpl) Desembaracar(ctx context.Context, input portsin.DesembaracarInput) (*portsin.Visao, error) {
	return u.transicionar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao, domainimportacao.AcaoDesembaracada,
		func(importacao *domainimportacao.Importacao, agora time.Time) (string, error) {
			err := importacao.Desembaracar(domainimportacao.DesembaracoInput{
				ChegadaEm:       input.ChegadaEm,
				DesembaracadaEm: input.DesembaracadaEm,
				Declaracao:      input.Declaracao,
			}, agora)
			if err != nil {
				return "", err
			}

			return "declaração " + *importacao.Declaracao, nil
		})
}

func (u *ImportacaoUsecaseImpl) Cancelar(ctx context.Context, input portsin.CancelarImportacaoInput) (*portsin.Visao, error) {
	return u.transicionar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao, domainimportacao.AcaoCancelada,
		func(importacao *domainimportacao.Importacao, agora time.Time) (string, error) {
			if err := importacao.Cancelar(input.Motivo, agora); err != nil {
				return "", err
			}

			return *importacao.CanceladaMotivo, nil
		})
}

func (u *ImportacaoUsecaseImpl) ListarHistorico(ctx context.Context, importacaoID, usuarioID uuid.UUID) ([]*domainimportacao.Historico, error) {
	perfil, err := u.exigeLeitura(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	if _, err := u.importacaoVisivel(ctx, perfil, importacaoID); err != nil {
		return nil, err
	}

	historicos, err := u.importacoes.ListarHistorico(ctx, importacaoID)
	if err != nil || perfil.financeiro {
		return historicos, err
	}

	semValores := make([]*domainimportacao.Historico, 0, len(historicos))
	for _, historico := range historicos {
		copia := *historico
		if copia.Acao.ExpoeValores() {
			copia.Detalhe = ""
		}
		semValores = append(semValores, &copia)
	}

	return semValores, nil
}

func (u *ImportacaoUsecaseImpl) transicionar(
	ctx context.Context,
	usuarioID, importacaoID uuid.UUID,
	versao int,
	acao domainimportacao.Acao,
	mudar func(*domainimportacao.Importacao, time.Time) (string, error),
) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, importacaoID)
	if err != nil {
		return nil, err
	}
	if err := importacao.ConferirVersao(versao); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	anterior := importacao.Status
	detalhe, err := mudar(importacao, agora)
	if err != nil {
		return nil, err
	}

	gravacao := &portsout.Gravacao{
		Importacao:     importacao,
		VersaoLida:     versao,
		StatusAnterior: anterior,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, usuarioID, acao, &anterior, importacao.Status, detalhe, agora),
		},
	}
	if importacao.Status == domainimportacao.StatusCancelada {
		gravacao.ContasExigidasEncerradas = importacao.ContasAPagarVinculadas()
	}
	if err := u.importacoes.Gravar(ctx, gravacao); err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: true}, nil
}

func (u *ImportacaoUsecaseImpl) exigirEmpresaAtiva(ctx context.Context, perfil perfilDoUsuario, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa da importação é obrigatória")
	}
	if !perfil.acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domainimportacao.ErroEmpresaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainimportacao.ErroEmpresaInativa()
	}

	return nil
}

func (u *ImportacaoUsecaseImpl) dadosValidados(
	ctx context.Context, atual *domainimportacao.Importacao, input portsin.DadosImportacaoInput,
) (domainimportacao.DadosImportacao, error) {
	moeda, err := u.moedaValidada(ctx, atual, input)
	if err != nil {
		return domainimportacao.DadosImportacao{}, err
	}
	itens, err := u.itensValidados(ctx, atual, input.Itens)
	if err != nil {
		return domainimportacao.DadosImportacao{}, err
	}

	return domainimportacao.DadosImportacao{
		FornecedorID:     input.FornecedorID,
		Moeda:            moeda,
		Referencia:       input.Referencia,
		EmbarquePrevisto: input.EmbarquePrevisto,
		ChegadaPrevista:  input.ChegadaPrevista,
		Observacoes:      input.Observacoes,
		Itens:            itens,
	}, nil
}

func (u *ImportacaoUsecaseImpl) moedaValidada(
	ctx context.Context, atual *domainimportacao.Importacao, input portsin.DadosImportacaoInput,
) (domainfornecedor.Moeda, error) {
	if input.FornecedorID == uuid.Nil {
		return "", domain.ErroValidacao("fornecedor da importação é obrigatório")
	}

	mesmoFornecedor := atual != nil && atual.FornecedorID == input.FornecedorID
	var padrao domainfornecedor.Moeda
	if mesmoFornecedor {
		padrao = atual.Moeda
	} else {
		fornecedor, err := u.fornecedores.Obter(ctx, input.FornecedorID)
		if err != nil {
			return "", err
		}
		if fornecedor == nil {
			return "", domainimportacao.ErroFornecedorNaoEncontrado()
		}
		if !fornecedor.Ativo {
			return "", domainimportacao.ErroFornecedorInativo(fornecedor.Nome)
		}
		padrao = fornecedor.MoedaPadrao
	}

	if input.Moeda == "" {
		return padrao, nil
	}

	return domainfornecedor.NovaMoeda(input.Moeda)
}

func (u *ImportacaoUsecaseImpl) itensValidados(
	ctx context.Context, atual *domainimportacao.Importacao, itens []portsin.ItemInput,
) ([]domainimportacao.ItemInput, error) {
	jaNaImportacao := make(map[uuid.UUID]bool)
	if atual != nil {
		for _, item := range atual.Itens {
			jaNaImportacao[item.ProdutoID] = true
		}
	}

	validados := make([]domainimportacao.ItemInput, 0, len(itens))
	conferidos := make(map[uuid.UUID]bool, len(itens))
	for _, item := range itens {
		if item.ProdutoID != uuid.Nil && !jaNaImportacao[item.ProdutoID] && !conferidos[item.ProdutoID] {
			if err := u.exigirProdutoAtivo(ctx, item.ProdutoID); err != nil {
				return nil, err
			}
			conferidos[item.ProdutoID] = true
		}

		validados = append(validados, domainimportacao.ItemInput{
			ProdutoID:            item.ProdutoID,
			Quantidade:           item.Quantidade,
			ValorMoedaCentesimos: item.ValorMoedaCentesimos,
		})
	}

	return validados, nil
}

func (u *ImportacaoUsecaseImpl) exigirProdutoAtivo(ctx context.Context, produtoID uuid.UUID) error {
	produto, err := u.produtos.Obter(ctx, produtoID)
	if err != nil {
		return err
	}
	if produto == nil {
		return domain.ErroValidacao("produto da importação não encontrado")
	}
	if !produto.Ativo {
		return domain.ErroValidacao(fmt.Sprintf("produto %q está inativo e não entra em importação nova", produto.Nome))
	}

	return nil
}
