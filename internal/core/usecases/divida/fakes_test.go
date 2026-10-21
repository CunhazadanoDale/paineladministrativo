package divida_test

import (
	"context"
	"sort"
	"time"

	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	usecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/divida"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios     map[uuid.UUID]*domainusuarios.Usuario
	cargos       map[uuid.UUID]*domainusuarios.Cargo
	empresas     map[uuid.UUID]*domainempresa.Empresa
	acessos      map[uuid.UUID]domainempresa.Acesso
	fornecedores map[uuid.UUID]*domainfornecedor.Fornecedor
	categorias   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	contas       map[uuid.UUID]*domainfinanceiro.ContaBancaria
	dividas      map[uuid.UUID]*domaindivida.Divida
	solicitacoes map[uuid.UUID]*domainsolicitacao.Solicitacao
	canceladas   []portsout.ContaCancelada
	historicos   []*domaindivida.Historico
	gravacoes    int
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:     map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:       map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:     map[uuid.UUID]*domainempresa.Empresa{},
		acessos:      map[uuid.UUID]domainempresa.Acesso{},
		fornecedores: map[uuid.UUID]*domainfornecedor.Fornecedor{},
		categorias:   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira{},
		contas:       map[uuid.UUID]*domainfinanceiro.ContaBancaria{},
		dividas:      map[uuid.UUID]*domaindivida.Divida{},
		solicitacoes: map[uuid.UUID]*domainsolicitacao.Solicitacao{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

func copiarDivida(divida *domaindivida.Divida) *domaindivida.Divida {
	copia := *divida
	copia.Parcelas = make([]*domaindivida.Parcela, 0, len(divida.Parcelas))
	for _, parcela := range divida.Parcelas {
		nova := *parcela
		nova.ContaAmortizacao = copiar(parcela.ContaAmortizacao)
		nova.ContaJuros = copiar(parcela.ContaJuros)
		copia.Parcelas = append(copia.Parcelas, &nova)
	}

	return &copia
}

func (m *memoria) usecase() *usecases.DividaUsecaseImpl {
	return usecases.NewDividaUsecase(
		repoDividas{m}, repoFornecedores{m}, repoCategorias{m}, repoContas{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
	)
}

func (m *memoria) usuario(financeiro bool, empresas ...uuid.UUID) uuid.UUID {
	cargo := &domainusuarios.Cargo{ID: uuid.New(), Financeiro: financeiro}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), CargoID: cargo.ID, Ativo: true}
	m.cargos[cargo.ID] = cargo
	m.usuarios[usuario.ID] = usuario
	m.acessos[usuario.ID] = domainempresa.AcessoRestrito(empresas)

	return usuario.ID
}

func (m *memoria) empresa(ativa bool) uuid.UUID {
	empresa := &domainempresa.Empresa{ID: uuid.New(), Ativa: ativa}
	m.empresas[empresa.ID] = empresa

	return empresa.ID
}

func (m *memoria) credor(ativo bool) uuid.UUID {
	fornecedor := &domainfornecedor.Fornecedor{ID: uuid.New(), Nome: "Banco Sicredi", Ativo: ativo}
	m.fornecedores[fornecedor.ID] = fornecedor

	return fornecedor.ID
}

func (m *memoria) categoria(linha domainfinanceiro.LinhaDRE) uuid.UUID {
	pai := uuid.New()
	categoria := &domainfinanceiro.CategoriaFinanceira{
		ID: uuid.New(), CategoriaPaiID: &pai, Natureza: domainfinanceiro.NaturezaDespesa, LinhaDRE: linha, Ativa: true,
	}
	m.categorias[categoria.ID] = categoria

	return categoria.ID
}

type repoUsuarios struct{ *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return copiar(r.usuarios[id]), nil
}

type repoCargos struct{ *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return copiar(r.cargos[id]), nil
}

type repoEmpresas struct{ *memoria }

func (r repoEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	return copiar(r.empresas[id]), nil
}

func (r repoEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return r.acessos[usuarioID], nil
}

type repoFornecedores struct{ *memoria }

func (r repoFornecedores) Obter(_ context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	return copiar(r.fornecedores[id]), nil
}

type repoCategorias struct{ *memoria }

func (r repoCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return copiar(r.categorias[id]), nil
}

type repoContas struct{ *memoria }

func (r repoContas) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	return copiar(r.contas[id]), nil
}

type repoDividas struct{ *memoria }

func (r repoDividas) Gravar(_ context.Context, gravacao *portsout.Gravacao) error {
	divida := gravacao.Divida
	if !gravacao.Novo {
		atual := r.dividas[divida.ID]
		if atual == nil {
			return domaindivida.ErroDividaNaoEncontrada()
		}
		if atual.Versao != gravacao.VersaoLida || atual.Status != gravacao.StatusAnterior {
			return domaindivida.ErroVersaoDivergente()
		}
		divida.Versao = atual.Versao + 1
	}

	if gravacao.ConferirContas {
		anteriores := map[uuid.UUID]domainsolicitacao.Status{}
		for _, conta := range gravacao.ContasCanceladas {
			anteriores[conta.SolicitacaoID] = conta.StatusAnterior
		}
		for _, parcela := range divida.Parcelas {
			for _, conta := range parcela.Contas() {
				esperado, cancelada := anteriores[conta.SolicitacaoID]
				if !cancelada {
					esperado = conta.Status
				}
				if r.solicitacoes[conta.SolicitacaoID].Status != esperado {
					return domaindivida.ErroContaAPagarAlterada()
				}
			}
		}
	}
	for _, conta := range gravacao.ContasCanceladas {
		solicitacao := r.solicitacoes[conta.SolicitacaoID]
		if solicitacao == nil || solicitacao.Status != conta.StatusAnterior {
			return domaindivida.ErroContaAPagarAlterada()
		}
		solicitacao.Status = domainsolicitacao.StatusCancelado
	}
	for _, conta := range gravacao.ContasAPagar {
		if fornecedor := r.fornecedores[*conta.Solicitacao.FornecedorID]; fornecedor == nil || !fornecedor.Ativo {
			return domainsolicitacao.ErroFornecedorInativo()
		}
	}
	for _, conta := range gravacao.ContasAPagar {
		r.solicitacoes[conta.Solicitacao.ID] = conta.Solicitacao
	}

	r.canceladas = append(r.canceladas, gravacao.ContasCanceladas...)
	r.historicos = append(r.historicos, gravacao.Historicos...)
	r.dividas[divida.ID] = copiarDivida(divida)
	if gravacao.Substituta != nil {
		r.dividas[gravacao.Substituta.ID] = copiarDivida(gravacao.Substituta)
	}
	r.gravacoes++

	return nil
}

func (r repoDividas) Obter(_ context.Context, id uuid.UUID) (*domaindivida.Divida, error) {
	divida := r.dividas[id]
	if divida == nil {
		return nil, nil
	}

	copia := copiarDivida(divida)
	for _, parcela := range copia.Parcelas {
		for _, conta := range parcela.Contas() {
			if solicitacao := r.solicitacoes[conta.SolicitacaoID]; solicitacao != nil {
				conta.Status = solicitacao.Status
			}
		}
	}

	return copia, nil
}

func (r repoDividas) Listar(_ context.Context, filtro portsout.DividaFiltro) ([]*domaindivida.Divida, error) {
	permitidas := map[uuid.UUID]bool{}
	for _, id := range filtro.EmpresaIDs {
		permitidas[id] = true
	}

	resultado := make([]*domaindivida.Divida, 0)
	for _, divida := range r.dividas {
		switch {
		case filtro.Empresa != nil && divida.EmpresaID != *filtro.Empresa:
		case filtro.EmpresaIDs != nil && !permitidas[divida.EmpresaID]:
		case filtro.CredorTipo != "" && divida.CredorTipo != filtro.CredorTipo:
		case filtro.Status != "" && divida.Status != filtro.Status:
		default:
			resultado = append(resultado, copiarDivida(divida))
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].ID.String() < resultado[j].ID.String() })

	return resultado, nil
}

func (r repoDividas) ListarHistorico(_ context.Context, dividaID uuid.UUID) ([]*domaindivida.Historico, error) {
	historicos := make([]*domaindivida.Historico, 0)
	for _, historico := range r.historicos {
		if historico.DividaID == dividaID {
			historicos = append(historicos, historico)
		}
	}

	return historicos, nil
}

func (r repoDividas) ComParcelasAGerar(_ context.Context, limite time.Time, empresaIDs []uuid.UUID) ([]uuid.UUID, error) {
	permitidas := map[uuid.UUID]bool{}
	for _, id := range empresaIDs {
		permitidas[id] = true
	}

	ids := make([]uuid.UUID, 0)
	for _, divida := range r.dividas {
		if divida.Status != domaindivida.StatusAtiva || (empresaIDs != nil && !permitidas[divida.EmpresaID]) {
			continue
		}
		for _, parcela := range divida.Parcelas {
			if parcela.Status == domaindivida.StatusParcelaPrevista && !parcela.Vencimento.After(limite) {
				ids = append(ids, divida.ID)
				break
			}
		}
	}

	return ids, nil
}
