package tabelapreco

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	"github.com/google/uuid"
)

var _ portsin.ImportacaoUseCase = (*ImportacaoUsecaseImpl)(nil)

type ImportacaoUsecaseImpl struct {
	itens    portsout.TabelaPrecoImportacaoRepository
	produtos portsout.ProdutoImportacaoRepository
	permissoes
}

func NewImportacaoUsecase(
	itens portsout.TabelaPrecoImportacaoRepository,
	produtos portsout.ProdutoImportacaoRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ImportacaoUsecaseImpl {
	return &ImportacaoUsecaseImpl{itens: itens, produtos: produtos, permissoes: permissoes{usuarios: usuarios, cargos: cargos}}
}

type importacaoDeItens struct {
	usuarioID          uuid.UUID
	simular            bool
	agora              time.Time
	produtosPorCodigo  map[string]uuid.UUID
	produtos           map[uuid.UUID]*domainestoque.Produto
	vigenciasDoArquivo map[string]int
}

func (u *ImportacaoUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarItensInput) (*domain.RelatorioImportacao, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}
	relatorio, err := domain.NovoRelatorioImportacao(input.Simular, len(input.Linhas))
	if err != nil {
		return nil, err
	}

	estado := &importacaoDeItens{
		usuarioID:          input.UsuarioID,
		simular:            input.Simular,
		agora:              time.Now().UTC(),
		produtos:           map[uuid.UUID]*domainestoque.Produto{},
		vigenciasDoArquivo: map[string]int{},
	}
	if estado.produtosPorCodigo, err = u.produtos.IDsPorCodigos(ctx, codigosDasLinhas(input.Linhas)); err != nil {
		return nil, err
	}

	for _, linha := range input.Linhas {
		motivo, err := u.importarLinha(ctx, estado, linha)
		if err != nil {
			return nil, err
		}
		if motivo != "" {
			relatorio.Rejeitar(linha.Numero, strings.TrimSpace(linha.Codigo), motivo)
			continue
		}
		relatorio.Importadas++
	}

	return relatorio, nil
}

func (u *ImportacaoUsecaseImpl) importarLinha(ctx context.Context, estado *importacaoDeItens, linha portsin.LinhaImportacao) (string, error) {
	codigo := strings.TrimSpace(linha.Codigo)
	if codigo == "" {
		return "código do produto é obrigatório", nil
	}
	produtoID, encontrado := estado.produtosPorCodigo[codigo]
	if !encontrado {
		return "produto com código " + codigo + " não encontrado", nil
	}

	preco, err := domain.CentavosDeImportacao(linha.Preco, "preco")
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	if preco == nil {
		return "preco é obrigatório", nil
	}
	vigenteDe, err := domain.DataDeImportacao(linha.VigenteDe, "vigente_de")
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	if vigenteDe == nil {
		return "vigente_de é obrigatório", nil
	}

	item, err := domaintabelapreco.NovoItem(domaintabelapreco.DadosItem{
		ProdutoID:     produtoID,
		Modelo:        linha.Modelo,
		PrecoCentavos: *preco,
		VigenteDe:     *vigenteDe,
		Observacao:    linha.Observacao,
	}, estado.usuarioID, estado.agora)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	chave := produtoID.String() + "|" + string(item.Modelo) + "|" + item.VigenteDe.Format(time.DateOnly)
	if anterior, repetido := estado.vigenciasDoArquivo[chave]; repetido {
		return "produto, modelo e início repetidos no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
	}

	produto, err := u.produtoDaLinha(ctx, estado, produtoID)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	if !produto.Ativo {
		return domain.MotivoDeRejeicao(domaintabelapreco.ErroProdutoInativo())
	}

	existe, err := u.itens.ExisteVigencia(ctx, item.ProdutoID, item.Modelo, item.VigenteDe)
	if err != nil {
		return "", err
	}
	if existe {
		return domain.MotivoDeRejeicao(domaintabelapreco.ErroVigenciaRepetida())
	}

	if !estado.simular {
		if err := u.itens.Criar(ctx, item); err != nil {
			return domain.MotivoDeRejeicao(err)
		}
	}

	estado.vigenciasDoArquivo[chave] = linha.Numero

	return "", nil
}

func (u *ImportacaoUsecaseImpl) produtoDaLinha(ctx context.Context, estado *importacaoDeItens, produtoID uuid.UUID) (*domainestoque.Produto, error) {
	if produto, ok := estado.produtos[produtoID]; ok {
		return produto, nil
	}

	produto, err := u.produtos.Obter(ctx, produtoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, domaintabelapreco.ErroProdutoNaoEncontrado()
	}
	estado.produtos[produtoID] = produto

	return produto, nil
}

func codigosDasLinhas(linhas []portsin.LinhaImportacao) []string {
	codigos := make([]string, 0, len(linhas))
	for _, linha := range linhas {
		if codigo := strings.TrimSpace(linha.Codigo); codigo != "" {
			codigos = append(codigos, codigo)
		}
	}

	return codigos
}
