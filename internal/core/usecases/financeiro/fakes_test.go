package financeiro_test

import (
	"context"
	"slices"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/google/uuid"
)

var (
	_ portsout.ContaBancariaRepository       = (*repositorioContas)(nil)
	_ portsout.CategoriaFinanceiraRepository = (*repositorioCategorias)(nil)
	_ portsout.EmpresaRepository             = (*repositorioEmpresas)(nil)
	_ portsout.UsuarioRepository             = (*repositorioUsuarios)(nil)
	_ portsout.CargoRepository               = (*repositorioCargos)(nil)
)

type repositorioEmpresas struct {
	itens   map[uuid.UUID]*domainempresa.Empresa
	acessos map[uuid.UUID]domainempresa.Acesso
}

func novoRepositorioEmpresas() *repositorioEmpresas {
	return &repositorioEmpresas{
		itens:   map[uuid.UUID]*domainempresa.Empresa{},
		acessos: map[uuid.UUID]domainempresa.Acesso{},
	}
}

func (r *repositorioEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	empresa, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *empresa
	return &copia, nil
}

func (r *repositorioEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	if acesso, ok := r.acessos[usuarioID]; ok {
		return acesso, nil
	}

	return domainempresa.AcessoRestrito(nil), nil
}

type repositorioUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func (r *repositorioUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *usuario
	return &copia, nil
}

type repositorioCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func (r *repositorioCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *cargo
	return &copia, nil
}

type repositorioContas struct {
	itens        map[uuid.UUID]*domainfinanceiro.ContaBancaria
	empresas     *repositorioEmpresas
	ultimoFiltro portsout.ContaBancariaFiltro
	gravacoes    int
}

func (r *repositorioContas) Criar(_ context.Context, conta *domainfinanceiro.ContaBancaria) error {
	if err := r.exigirEmpresaAtiva(conta.EmpresaID); err != nil {
		return err
	}
	if err := r.conferirUnicidade(conta); err != nil {
		return err
	}

	copia := *conta
	r.itens[conta.ID] = &copia
	r.gravacoes++

	return nil
}

func (r *repositorioContas) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	conta, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *conta
	return &copia, nil
}

func (r *repositorioContas) Listar(_ context.Context, filtro portsout.ContaBancariaFiltro) ([]*domainfinanceiro.ContaBancaria, error) {
	r.ultimoFiltro = filtro

	contas := make([]*domainfinanceiro.ContaBancaria, 0, len(r.itens))
	for _, conta := range r.itens {
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, conta.EmpresaID) {
			continue
		}
		if filtro.Banco != nil && conta.Banco != *filtro.Banco {
			continue
		}
		if filtro.Tipo != nil && conta.Tipo != *filtro.Tipo {
			continue
		}
		if filtro.Ativa != nil && conta.Ativa != *filtro.Ativa {
			continue
		}
		copia := *conta
		contas = append(contas, &copia)
	}

	slices.SortFunc(contas, func(a, b *domainfinanceiro.ContaBancaria) int {
		if comparacao := strings.Compare(strings.ToLower(a.Apelido), strings.ToLower(b.Apelido)); comparacao != 0 {
			return comparacao
		}
		return strings.Compare(a.ID.String(), b.ID.String())
	})

	return contas, nil
}

func (r *repositorioContas) Atualizar(_ context.Context, conta *domainfinanceiro.ContaBancaria) error {
	return r.gravar(conta)
}

func (r *repositorioContas) Ativar(_ context.Context, conta *domainfinanceiro.ContaBancaria) error {
	if err := r.exigirEmpresaAtiva(conta.EmpresaID); err != nil {
		return err
	}

	return r.gravar(conta)
}

func (r *repositorioContas) Desativar(_ context.Context, conta *domainfinanceiro.ContaBancaria) error {
	return r.gravar(conta)
}

func (r *repositorioContas) Remover(_ context.Context, id uuid.UUID) error {
	if _, ok := r.itens[id]; !ok {
		return domainfinanceiro.ErroContaBancariaNaoEncontrada()
	}

	delete(r.itens, id)

	return nil
}

func (r *repositorioContas) gravar(conta *domainfinanceiro.ContaBancaria) error {
	existente, ok := r.itens[conta.ID]
	if !ok {
		return domainfinanceiro.ErroContaBancariaNaoEncontrada()
	}
	if existente.Versao != conta.Versao {
		return domainfinanceiro.ErroContaBancariaAlterada()
	}
	if err := r.conferirUnicidade(conta); err != nil {
		return err
	}

	conta.Versao++
	copia := *conta
	r.itens[conta.ID] = &copia
	r.gravacoes++

	return nil
}

func (r *repositorioContas) exigirEmpresaAtiva(empresaID uuid.UUID) error {
	empresa, ok := r.empresas.itens[empresaID]
	if !ok {
		return domainfinanceiro.ErroEmpresaDaContaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainfinanceiro.ErroEmpresaInativa()
	}

	return nil
}

func (r *repositorioContas) conferirUnicidade(conta *domainfinanceiro.ContaBancaria) error {
	for _, existente := range r.itens {
		if existente.ID == conta.ID {
			continue
		}
		mesmaAgencia := existente.Banco == conta.Banco && existente.Tipo == conta.Tipo &&
			semZeros(existente.Agencia.Numero()) == semZeros(conta.Agencia.Numero())
		mesmoNumero := semZeros(existente.Conta.Numero()) == semZeros(conta.Conta.Numero()) ||
			semZeros(comDigito(existente.Conta)) == semZeros(comDigito(conta.Conta))
		if mesmaAgencia && mesmoNumero {
			return domainfinanceiro.ErroContaDuplicada()
		}
		if existente.EmpresaID == conta.EmpresaID && strings.EqualFold(existente.Apelido, conta.Apelido) {
			return domainfinanceiro.ErroApelidoRepetido()
		}
	}

	return nil
}

func semZeros(numero string) string {
	return strings.TrimLeft(numero, "0")
}

func comDigito(numero domainfinanceiro.NumeroBancario) string {
	if digito := numero.Digito(); digito != nil {
		return numero.Numero() + *digito
	}

	return numero.Numero()
}

type repositorioCategorias struct {
	itens        map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	ultimoFiltro portsout.CategoriaFinanceiraFiltro
	gravacoes    int
}

func (r *repositorioCategorias) Criar(_ context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	if categoria.EhSubcategoria() {
		pai, ok := r.itens[*categoria.CategoriaPaiID]
		if !ok || pai.EhSubcategoria() || !pai.Ativa {
			return domainfinanceiro.ErroPaiDeveSerRaizAtiva()
		}
		categoria.Natureza = pai.Natureza
		categoria.LinhaDRE = pai.LinhaDRE
	}
	if err := r.conferirUnicidade(categoria); err != nil {
		return err
	}

	copia := *categoria
	r.itens[categoria.ID] = &copia
	r.gravacoes++

	return nil
}

func (r *repositorioCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	categoria, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *categoria
	return &copia, nil
}

func (r *repositorioCategorias) Listar(_ context.Context, filtro portsout.CategoriaFinanceiraFiltro) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	r.ultimoFiltro = filtro

	categorias := make([]*domainfinanceiro.CategoriaFinanceira, 0, len(r.itens))
	for _, categoria := range r.ordenadas() {
		if filtro.Natureza != nil && categoria.Natureza != *filtro.Natureza {
			continue
		}
		if filtro.LinhaDRE != nil && categoria.LinhaDRE != *filtro.LinhaDRE {
			continue
		}
		if filtro.Ativa != nil && categoria.Ativa != *filtro.Ativa {
			continue
		}
		categorias = append(categorias, categoria)
	}

	return categorias, nil
}

func (r *repositorioCategorias) ListarPlano(_ context.Context, somenteAtivas bool) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	categorias := make([]*domainfinanceiro.CategoriaFinanceira, 0, len(r.itens))
	for _, categoria := range r.ordenadas() {
		if somenteAtivas && !categoria.Ativa {
			continue
		}
		categorias = append(categorias, categoria)
	}

	return categorias, nil
}

func (r *repositorioCategorias) PossuiSubcategoriasAtivas(_ context.Context, id uuid.UUID) (bool, error) {
	for _, categoria := range r.itens {
		if categoria.CategoriaPaiID != nil && *categoria.CategoriaPaiID == id && categoria.Ativa {
			return true, nil
		}
	}

	return false, nil
}

func (r *repositorioCategorias) Atualizar(_ context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	return r.gravar(categoria)
}

func (r *repositorioCategorias) Ativar(_ context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	if categoria.EhSubcategoria() {
		if pai, ok := r.itens[*categoria.CategoriaPaiID]; !ok || !pai.Ativa {
			return domainfinanceiro.ErroPaiInativo()
		}
	}

	return r.gravar(categoria)
}

func (r *repositorioCategorias) Desativar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	if !categoria.EhSubcategoria() {
		if possui, _ := r.PossuiSubcategoriasAtivas(ctx, categoria.ID); possui {
			return domainfinanceiro.ErroSubcategoriasAtivas()
		}
	}

	return r.gravar(categoria)
}

func (r *repositorioCategorias) Remover(_ context.Context, id uuid.UUID) error {
	if _, ok := r.itens[id]; !ok {
		return domainfinanceiro.ErroCategoriaFinanceiraNaoEncontrada()
	}
	for _, categoria := range r.itens {
		if categoria.CategoriaPaiID != nil && *categoria.CategoriaPaiID == id {
			return domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
		}
	}

	delete(r.itens, id)

	return nil
}

func (r *repositorioCategorias) gravar(categoria *domainfinanceiro.CategoriaFinanceira) error {
	existente, ok := r.itens[categoria.ID]
	if !ok {
		return domainfinanceiro.ErroCategoriaFinanceiraNaoEncontrada()
	}
	if existente.Versao != categoria.Versao {
		return domainfinanceiro.ErroCategoriaFinanceiraAlterada()
	}
	if err := r.conferirUnicidade(categoria); err != nil {
		return err
	}

	categoria.Versao++
	copia := *categoria
	r.itens[categoria.ID] = &copia
	r.gravacoes++

	return nil
}

func (r *repositorioCategorias) conferirUnicidade(categoria *domainfinanceiro.CategoriaFinanceira) error {
	for _, existente := range r.itens {
		if existente.ID == categoria.ID {
			continue
		}
		mesmoNivel := (existente.CategoriaPaiID == nil && categoria.CategoriaPaiID == nil) ||
			(existente.CategoriaPaiID != nil && categoria.CategoriaPaiID != nil && *existente.CategoriaPaiID == *categoria.CategoriaPaiID)
		if mesmoNivel && strings.EqualFold(existente.Nome, categoria.Nome) {
			return domainfinanceiro.ErroNomeDeCategoriaRepetido()
		}
		if existente.Codigo != nil && categoria.Codigo != nil && *existente.Codigo == *categoria.Codigo {
			return domainfinanceiro.ErroCodigoDeCategoriaRepetido()
		}
	}

	return nil
}

func (r *repositorioCategorias) ordenadas() []*domainfinanceiro.CategoriaFinanceira {
	categorias := make([]*domainfinanceiro.CategoriaFinanceira, 0, len(r.itens))
	for _, categoria := range r.itens {
		copia := *categoria
		categorias = append(categorias, &copia)
	}

	slices.SortFunc(categorias, func(a, b *domainfinanceiro.CategoriaFinanceira) int {
		if a.Ordem != b.Ordem {
			return a.Ordem - b.Ordem
		}
		if comparacao := strings.Compare(strings.ToLower(a.Nome), strings.ToLower(b.Nome)); comparacao != 0 {
			return comparacao
		}
		return strings.Compare(a.ID.String(), b.ID.String())
	})

	return categorias
}
