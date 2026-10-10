package estoque_test

import (
	"context"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	portsoutsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var (
	_ portsout.CategoriaRepository          = (*repositorioCategorias)(nil)
	_ portsout.ProdutoRepository            = (*repositorioProdutos)(nil)
	_ portsout.MovimentoRepository          = (*repositorioMovimentos)(nil)
	_ portsout.ImagemRepository             = (*repositorioImagens)(nil)
	_ portsoutsolicitacao.ArquivoRepository = (*repositorioArquivos)(nil)
	_ portsoutusuarios.UsuarioRepository    = (*repositorioUsuarios)(nil)
	_ portsoutusuarios.CargoRepository      = (*repositorioCargos)(nil)
)

type repositorioCategorias struct {
	itens map[uuid.UUID]*domainestoque.Categoria
}

func novoRepositorioCategorias() *repositorioCategorias {
	return &repositorioCategorias{itens: map[uuid.UUID]*domainestoque.Categoria{}}
}

func (r *repositorioCategorias) Criar(_ context.Context, categoria *domainestoque.Categoria) (uuid.UUID, error) {
	copia := *categoria
	r.itens[categoria.ID] = &copia

	return categoria.ID, nil
}

func (r *repositorioCategorias) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Categoria, error) {
	categoria, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *categoria
	return &copia, nil
}

func (r *repositorioCategorias) ObterPorSlug(_ context.Context, slug string) (*domainestoque.Categoria, error) {
	for _, categoria := range r.itens {
		if categoria.Slug.Valor() == slug {
			copia := *categoria
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioCategorias) Listar(_ context.Context, filtro portsout.CategoriaFiltro) ([]*domainestoque.Categoria, error) {
	itens := make([]*domainestoque.Categoria, 0, len(r.itens))
	for _, categoria := range r.itens {
		if filtro.Ativo != nil && categoria.Ativo != *filtro.Ativo {
			continue
		}
		copia := *categoria
		itens = append(itens, &copia)
	}

	paginacao := filtro.Normalizada()
	inicio := (paginacao.Page - 1) * paginacao.Size
	if inicio >= len(itens) {
		return []*domainestoque.Categoria{}, nil
	}

	fim := inicio + paginacao.Size
	if fim > len(itens) {
		fim = len(itens)
	}

	return itens[inicio:fim], nil
}

func (r *repositorioCategorias) Atualizar(_ context.Context, categoria *domainestoque.Categoria) (bool, error) {
	if _, ok := r.itens[categoria.ID]; !ok {
		return false, nil
	}

	copia := *categoria
	r.itens[categoria.ID] = &copia

	return true, nil
}

func (r *repositorioCategorias) PossuiSubcategorias(_ context.Context, id uuid.UUID) (bool, error) {
	for _, categoria := range r.itens {
		if categoria.CategoriaPaiID != nil && *categoria.CategoriaPaiID == id {
			return true, nil
		}
	}

	return false, nil
}

func (r *repositorioCategorias) PossuiProdutos(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

type repositorioProdutos struct {
	categorias *repositorioCategorias
	itens      map[uuid.UUID]*domainestoque.Produto
	movimentos map[uuid.UUID][]*domainestoque.Movimento
}

func novoRepositorioProdutos(categorias *repositorioCategorias) *repositorioProdutos {
	return &repositorioProdutos{
		categorias: categorias,
		itens:      map[uuid.UUID]*domainestoque.Produto{},
		movimentos: map[uuid.UUID][]*domainestoque.Movimento{},
	}
}

func (r *repositorioProdutos) Criar(_ context.Context, produto *domainestoque.Produto) (uuid.UUID, error) {
	copia := *produto
	r.itens[produto.ID] = &copia

	return produto.ID, nil
}

func (r *repositorioProdutos) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	produto, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *produto
	return &copia, nil
}

func (r *repositorioProdutos) ObterPorSlug(_ context.Context, slug string) (*domainestoque.Produto, error) {
	for _, produto := range r.itens {
		if produto.Slug.Valor() == slug {
			copia := *produto
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioProdutos) Listar(_ context.Context, filtro portsout.ProdutoFiltro) ([]*domainestoque.Produto, error) {
	itens := make([]*domainestoque.Produto, 0, len(r.itens))
	for _, produto := range r.itens {
		if filtro.CategoriaID != nil && !bateCategoria(r.categorias, produto.CategoriaID, *filtro.CategoriaID) {
			continue
		}
		if filtro.Ativo != nil && produto.Ativo != *filtro.Ativo {
			continue
		}
		if filtro.Destaque != nil && produto.Destaque != *filtro.Destaque {
			continue
		}
		if filtro.EstoqueBaixo && !produto.EstoqueBaixo() {
			continue
		}
		if filtro.ComSaldo && produto.Saldo.Vazio() {
			continue
		}
		if filtro.Busca != "" && !bateBusca(produto, filtro.Busca) {
			continue
		}
		if filtro.NaVitrine && !categoriaNaVitrine(r.categorias, produto.CategoriaID) {
			continue
		}
		copia := *produto
		itens = append(itens, &copia)
	}

	return itens, nil
}

func categoriaNaVitrine(categorias *repositorioCategorias, categoriaID uuid.UUID) bool {
	categoria, ok := categorias.itens[categoriaID]
	if !ok || !categoria.Ativo {
		return false
	}
	if categoria.CategoriaPaiID == nil {
		return true
	}

	pai, ok := categorias.itens[*categoria.CategoriaPaiID]

	return ok && pai.Ativo
}

func bateCategoria(categorias *repositorioCategorias, categoriaProduto, categoriaFiltro uuid.UUID) bool {
	if categoriaProduto == categoriaFiltro {
		return true
	}

	categoria, ok := categorias.itens[categoriaProduto]
	if !ok || categoria.CategoriaPaiID == nil {
		return false
	}

	return *categoria.CategoriaPaiID == categoriaFiltro
}

func bateBusca(produto *domainestoque.Produto, busca string) bool {
	busca = strings.ToLower(busca)

	if strings.Contains(strings.ToLower(produto.Nome), busca) {
		return true
	}
	if produto.Codigo != nil && strings.Contains(strings.ToLower(*produto.Codigo), busca) {
		return true
	}

	return false
}

func (r *repositorioProdutos) Resumo(_ context.Context) (domainestoque.ResumoProdutos, error) {
	var resumo domainestoque.ResumoProdutos

	for _, produto := range r.itens {
		resumo.TotalProdutos++

		if !produto.Ativo {
			continue
		}

		resumo.ProdutosAtivos++

		if produto.Preco != nil {
			resumo.ValorEstoqueCentavos += produto.Preco.Centavos() * int64(produto.Saldo.Quantidade())
		}
		if produto.EstoqueBaixo() {
			resumo.ProdutosEstoqueBaixo++
		}
	}

	return resumo, nil
}

func (r *repositorioProdutos) Atualizar(_ context.Context, produto *domainestoque.Produto) (bool, error) {
	if _, ok := r.itens[produto.ID]; !ok {
		return false, nil
	}

	copia := *produto
	r.itens[produto.ID] = &copia

	return true, nil
}

func (r *repositorioProdutos) Movimentar(_ context.Context, produto *domainestoque.Produto, movimento *domainestoque.Movimento) error {
	salvo, ok := r.itens[produto.ID]
	if !ok {
		return domain.ErroNaoEncontrado("produto não encontrado")
	}

	if salvo.Saldo.Quantidade() != movimento.SaldoApos.Quantidade()-movimento.Tipo.Efeito(movimento.Quantidade) {
		return domain.ErroConflito("produto com saldo alterado por outra operação, recarregue e tente novamente")
	}

	copia := *produto
	r.itens[produto.ID] = &copia
	r.movimentos[produto.ID] = append(r.movimentos[produto.ID], movimento)

	return nil
}

type repositorioMovimentos struct {
	origem *repositorioProdutos
}

func novoRepositorioMovimentos(origem *repositorioProdutos) *repositorioMovimentos {
	return &repositorioMovimentos{origem: origem}
}

func (r *repositorioMovimentos) Listar(_ context.Context, filtro portsout.MovimentoFiltro) ([]*domainestoque.Movimento, error) {
	itens := make([]*domainestoque.Movimento, 0)
	for _, movimentos := range r.origem.movimentos {
		for _, movimento := range movimentos {
			if filtro.ProdutoID != nil && movimento.ProdutoID != *filtro.ProdutoID {
				continue
			}
			if filtro.Tipo != "" && movimento.Tipo != filtro.Tipo {
				continue
			}
			copia := *movimento
			itens = append(itens, &copia)
		}
	}

	sort.Slice(itens, func(i, j int) bool {
		return itens[i].CriadoEm.After(itens[j].CriadoEm)
	})

	return itens, nil
}

func (r *repositorioMovimentos) ListarResumo(_ context.Context, limite int) ([]*domainestoque.MovimentoResumo, error) {
	itens := make([]domainestoque.MovimentoResumo, 0)
	for _, movimentos := range r.origem.movimentos {
		for _, movimento := range movimentos {
			produto, ok := r.origem.itens[movimento.ProdutoID]
			if !ok {
				continue
			}

			itens = append(itens, domainestoque.MovimentoResumo{
				ID:          movimento.ID,
				ProdutoID:   movimento.ProdutoID,
				ProdutoNome: produto.Nome,
				Tipo:        movimento.Tipo,
				Quantidade:  movimento.Quantidade,
				SaldoApos:   movimento.SaldoApos,
				CriadoEm:    movimento.CriadoEm,
			})
		}
	}

	sort.Slice(itens, func(i, j int) bool {
		return itens[i].CriadoEm.After(itens[j].CriadoEm)
	})

	if limite > 0 && len(itens) > limite {
		itens = itens[:limite]
	}

	resumo := make([]*domainestoque.MovimentoResumo, 0, len(itens))
	for i := range itens {
		resumo = append(resumo, &itens[i])
	}

	return resumo, nil
}

type repositorioUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func novoRepositorioUsuarios() *repositorioUsuarios {
	return &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
}

func (r *repositorioUsuarios) Create(_ context.Context, usuario *domainusuarios.Usuario) (uuid.UUID, error) {
	copia := *usuario
	r.itens[usuario.ID] = &copia

	return usuario.ID, nil
}

func (r *repositorioUsuarios) Update(_ context.Context, usuario *domainusuarios.Usuario) error {
	copia := *usuario
	r.itens[usuario.ID] = &copia

	return nil
}

func (r *repositorioUsuarios) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *usuario
	return &copia, nil
}

func (r *repositorioUsuarios) GetByEmail(_ context.Context, email string) (*domainusuarios.Usuario, error) {
	for _, usuario := range r.itens {
		if usuario.Email == email {
			copia := *usuario
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioUsuarios) List(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return nil, nil
}

func (r *repositorioUsuarios) ListAtivos(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return nil, nil
}

func (r *repositorioUsuarios) Search(_ context.Context, _ string, _ domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return nil, nil
}

func (r *repositorioUsuarios) UpdateUltimoLogin(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (r *repositorioUsuarios) AtualizarSenha(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (r *repositorioUsuarios) EncerrarSessoes(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *repositorioUsuarios) ContarAdministradoresAtivos(_ context.Context) (int, error) {
	return 0, nil
}

func (r *repositorioUsuarios) ContarAtivosPorCargo(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *repositorioUsuarios) Ativar(_ context.Context, id uuid.UUID) error {
	if usuario, ok := r.itens[id]; ok {
		usuario.Ativo = true
	}

	return nil
}

func (r *repositorioUsuarios) Desativar(_ context.Context, id uuid.UUID) error {
	if usuario, ok := r.itens[id]; ok {
		usuario.Ativo = false
	}

	return nil
}

func (r *repositorioUsuarios) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

type repositorioCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func novoRepositorioCargos() *repositorioCargos {
	return &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
}

func (r *repositorioCargos) Create(_ context.Context, cargo *domainusuarios.Cargo) (uuid.UUID, error) {
	copia := *cargo
	r.itens[cargo.ID] = &copia

	return cargo.ID, nil
}

func (r *repositorioCargos) Update(_ context.Context, cargo *domainusuarios.Cargo) error {
	copia := *cargo
	r.itens[cargo.ID] = &copia

	return nil
}

func (r *repositorioCargos) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *cargo
	return &copia, nil
}

func (r *repositorioCargos) GetByNome(_ context.Context, nome string) (*domainusuarios.Cargo, error) {
	for _, cargo := range r.itens {
		if cargo.Nome == nome {
			copia := *cargo
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioCargos) List(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return nil, nil
}

func (r *repositorioCargos) ListAtivos(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return nil, nil
}

func (r *repositorioCargos) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

type repositorioImagens struct {
	itens map[uuid.UUID]*domainestoque.Imagem
}

func novoRepositorioImagens() *repositorioImagens {
	return &repositorioImagens{itens: map[uuid.UUID]*domainestoque.Imagem{}}
}

func (r *repositorioImagens) Criar(_ context.Context, imagem *domainestoque.Imagem) (uuid.UUID, error) {
	copia := *imagem
	r.itens[imagem.ID] = &copia

	return imagem.ID, nil
}

func (r *repositorioImagens) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Imagem, error) {
	imagem, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *imagem
	return &copia, nil
}

func (r *repositorioImagens) ListarPorProduto(_ context.Context, produtoID uuid.UUID) ([]*domainestoque.Imagem, error) {
	return r.listar(func(imagem *domainestoque.Imagem) bool {
		return imagem.ProdutoID == produtoID
	}), nil
}

func (r *repositorioImagens) ListarPorProdutos(_ context.Context, produtoIDs []uuid.UUID) ([]*domainestoque.Imagem, error) {
	permitidos := make(map[uuid.UUID]bool, len(produtoIDs))
	for _, id := range produtoIDs {
		permitidos[id] = true
	}

	return r.listar(func(imagem *domainestoque.Imagem) bool {
		return permitidos[imagem.ProdutoID]
	}), nil
}

func (r *repositorioImagens) Remover(_ context.Context, id uuid.UUID) (bool, error) {
	if _, ok := r.itens[id]; !ok {
		return false, nil
	}

	delete(r.itens, id)

	return true, nil
}

func (r *repositorioImagens) ArquivoEmUso(_ context.Context, arquivoID uuid.UUID) (bool, error) {
	for _, imagem := range r.itens {
		if imagem.ArquivoID == arquivoID {
			return true, nil
		}
	}

	return false, nil
}

func (r *repositorioImagens) listar(pertence func(*domainestoque.Imagem) bool) []*domainestoque.Imagem {
	itens := make([]*domainestoque.Imagem, 0, len(r.itens))
	for _, imagem := range r.itens {
		if !pertence(imagem) {
			continue
		}

		copia := *imagem
		itens = append(itens, &copia)
	}

	sort.Slice(itens, func(i, j int) bool {
		if itens[i].Ordem != itens[j].Ordem {
			return itens[i].Ordem < itens[j].Ordem
		}
		if !itens[i].CriadoEm.Equal(itens[j].CriadoEm) {
			return itens[i].CriadoEm.Before(itens[j].CriadoEm)
		}

		return itens[i].ID.String() < itens[j].ID.String()
	})

	return itens
}

type repositorioArquivos struct {
	itens      map[uuid.UUID]*domainsolicitacao.Arquivo
	vinculados map[uuid.UUID]bool
}

func novoRepositorioArquivos() *repositorioArquivos {
	return &repositorioArquivos{itens: map[uuid.UUID]*domainsolicitacao.Arquivo{}, vinculados: map[uuid.UUID]bool{}}
}

func (r *repositorioArquivos) Criar(_ context.Context, arquivo *domainsolicitacao.Arquivo) (uuid.UUID, error) {
	copia := *arquivo
	r.itens[arquivo.ID] = &copia

	return arquivo.ID, nil
}

func (r *repositorioArquivos) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error) {
	arquivo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *arquivo
	return &copia, nil
}

func (r *repositorioArquivos) ListarPorProprietario(_ context.Context, _ uuid.UUID, _ domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error) {
	return nil, nil
}

func (r *repositorioArquivos) VinculadoASolicitacao(_ context.Context, id uuid.UUID) (bool, error) {
	return r.vinculados[id], nil
}

func (r *repositorioArquivos) SolicitacaoDoArquivo(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}

func (r *repositorioArquivos) Remover(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

type storageEmMemoria struct {
	removidas []string
}

func (s *storageEmMemoria) Enviar(_ context.Context, _ string, _ io.Reader, _ string) error {
	return nil
}

func (s *storageEmMemoria) Baixar(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (s *storageEmMemoria) Remover(_ context.Context, chave string) error {
	s.removidas = append(s.removidas, chave)

	return nil
}
