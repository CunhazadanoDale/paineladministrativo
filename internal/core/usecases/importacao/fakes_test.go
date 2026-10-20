package importacao_test

import (
	"context"
	"maps"
	"slices"
	"sort"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios           map[uuid.UUID]*domainusuarios.Usuario
	cargos             map[uuid.UUID]*domainusuarios.Cargo
	empresas           map[uuid.UUID]*domainempresa.Empresa
	acessos            map[uuid.UUID]domainempresa.Acesso
	fornecedores       map[uuid.UUID]*domainfornecedor.Fornecedor
	produtos           map[uuid.UUID]*domainestoque.Produto
	importacoes        map[uuid.UUID]*domainimportacao.Importacao
	historicos         []*domainimportacao.Historico
	sequencias         map[uuid.UUID]int
	gravacoes          []*portsout.Gravacao
	categorias         map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	contas             map[uuid.UUID]*domainsolicitacao.Solicitacao
	pedidoItens        map[uuid.UUID]*domainimportacao.PedidoItemAlocavel
	locais             map[uuid.UUID]*domainestoque.Local
	estoques           map[uuid.UUID]*domainestoque.EstoqueProduto
	reservas           map[uuid.UUID]*domainestoque.Reserva
	historicosDePedido []*domainpedido.Historico
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:     map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:       map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:     map[uuid.UUID]*domainempresa.Empresa{},
		acessos:      map[uuid.UUID]domainempresa.Acesso{},
		fornecedores: map[uuid.UUID]*domainfornecedor.Fornecedor{},
		produtos:     map[uuid.UUID]*domainestoque.Produto{},
		importacoes:  map[uuid.UUID]*domainimportacao.Importacao{},
		sequencias:   map[uuid.UUID]int{},
		categorias:   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira{},
		contas:       map[uuid.UUID]*domainsolicitacao.Solicitacao{},
		pedidoItens:  map[uuid.UUID]*domainimportacao.PedidoItemAlocavel{},
		locais:       map[uuid.UUID]*domainestoque.Local{},
		estoques:     map[uuid.UUID]*domainestoque.EstoqueProduto{},
		reservas:     map[uuid.UUID]*domainestoque.Reserva{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

func copiarImportacao(importacao *domainimportacao.Importacao) *domainimportacao.Importacao {
	copia := *importacao
	copia.Itens = make([]*domainimportacao.Item, 0, len(importacao.Itens))
	for _, item := range importacao.Itens {
		copia.Itens = append(copia.Itens, copiar(item))
	}
	copia.Parcelas = make([]*domainimportacao.Parcela, 0, len(importacao.Parcelas))
	for _, parcela := range importacao.Parcelas {
		copia.Parcelas = append(copia.Parcelas, copiar(parcela))
	}
	copia.Custos = make([]*domainimportacao.Custo, 0, len(importacao.Custos))
	for _, custo := range importacao.Custos {
		copia.Custos = append(copia.Custos, copiar(custo))
	}
	copia.Alocacoes = make([]*domainimportacao.Alocacao, 0, len(importacao.Alocacoes))
	for _, alocacao := range importacao.Alocacoes {
		copia.Alocacoes = append(copia.Alocacoes, copiar(alocacao))
	}

	return &copia
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

type repoProdutos struct{ *memoria }

func (r repoProdutos) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	return copiar(r.produtos[id]), nil
}

type repoImportacoes struct{ *memoria }

func (r repoImportacoes) Gravar(_ context.Context, gravacao *portsout.Gravacao) error {
	importacao := copiarImportacao(gravacao.Importacao)
	if gravacao.Novo {
		if empresa := r.empresas[importacao.EmpresaID]; empresa == nil || !empresa.Ativa {
			return domain.ErroValidacao("empresa inativa não recebe importação nova")
		}
		importacao.Numero = r.sequencias[importacao.EmpresaID] + 1
	} else {
		atual, ok := r.importacoes[importacao.ID]
		if !ok {
			return domain.ErroNaoEncontrado("importação não encontrada")
		}
		if atual.Versao != gravacao.VersaoLida || atual.Status != gravacao.StatusAnterior {
			return domainimportacao.ErroVersaoDivergente()
		}
		importacao.Versao = atual.Versao + 1
	}
	if gravacao.FornecedorTrocado {
		if fornecedor := r.fornecedores[importacao.FornecedorID]; fornecedor == nil || !fornecedor.Ativo {
			return domain.ErroValidacao("fornecedor inativo não recebe importação nova")
		}
	}
	for _, id := range gravacao.ContasExigidasEncerradas {
		if conta := r.contas[id]; conta == nil || (conta.Status != domainsolicitacao.StatusCancelado && conta.Status != domainsolicitacao.StatusRejeitado) {
			return domainimportacao.ErroContasAPagarAbertas()
		}
	}
	for _, conta := range gravacao.ContasAPagar {
		if _, existe := r.contas[conta.Solicitacao.ID]; existe {
			return domain.ErroConflito("conta a pagar já gerada")
		}
	}
	for _, pedidoID := range gravacao.PedidosAtendidos {
		for _, item := range r.pedidoItens {
			if item.PedidoID == pedidoID && !item.StatusPedido.EmAtendimento() {
				return domain.ErroConflito("pedido atendido saiu de atendimento")
			}
		}
	}
	estoques, reservas, err := r.aplicarOperacoes(gravacao.OperacoesEstoque)
	if err != nil {
		return err
	}

	if gravacao.Novo {
		r.sequencias[importacao.EmpresaID] = importacao.Numero
	}
	r.importacoes[importacao.ID] = importacao
	for id, estoque := range estoques {
		r.estoques[id] = estoque
	}
	for id, reserva := range reservas {
		r.reservas[id] = reserva
	}
	r.historicosDePedido = append(r.historicosDePedido, gravacao.HistoricosDePedido...)
	for _, conta := range gravacao.ContasAPagar {
		r.contas[conta.Solicitacao.ID] = copiar(conta.Solicitacao)
	}
	r.historicos = append(r.historicos, gravacao.Historicos...)
	r.gravacoes = append(r.gravacoes, gravacao)
	gravacao.Importacao.Numero = importacao.Numero
	gravacao.Importacao.Versao = importacao.Versao

	return nil
}

func (r repoImportacoes) Obter(_ context.Context, id uuid.UUID) (*domainimportacao.Importacao, error) {
	importacao, ok := r.importacoes[id]
	if !ok {
		return nil, nil
	}

	copia := copiarImportacao(importacao)
	for _, parcela := range copia.Parcelas {
		if parcela.SolicitacaoID != nil {
			status := r.contas[*parcela.SolicitacaoID].Status
			parcela.StatusContaAPagar = &status
		}
	}
	for _, custo := range copia.Custos {
		status := r.contas[custo.SolicitacaoID].Status
		custo.StatusContaAPagar = &status
	}

	return copia, nil
}

func (r repoImportacoes) Listar(_ context.Context, filtro portsout.ImportacaoFiltro) ([]*domainimportacao.Importacao, error) {
	encontradas := make([]*domainimportacao.Importacao, 0)
	for _, importacao := range r.importacoes {
		switch {
		case filtro.Empresa != nil && importacao.EmpresaID != *filtro.Empresa:
		case filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, importacao.EmpresaID):
		case filtro.FornecedorID != nil && importacao.FornecedorID != *filtro.FornecedorID:
		case filtro.Status != "" && importacao.Status != filtro.Status:
		case filtro.Numero != nil && importacao.Numero != *filtro.Numero:
		case filtro.ChegadaAte != nil && (importacao.ChegadaPrevista == nil || importacao.ChegadaPrevista.After(*filtro.ChegadaAte)):
		default:
			encontradas = append(encontradas, copiarImportacao(importacao))
		}
	}
	sort.Slice(encontradas, func(i, j int) bool { return encontradas[i].ID.String() < encontradas[j].ID.String() })

	return encontradas, nil
}

func (r repoImportacoes) ListarHistorico(_ context.Context, importacaoID uuid.UUID) ([]*domainimportacao.Historico, error) {
	historicos := make([]*domainimportacao.Historico, 0)
	for _, historico := range r.historicos {
		if historico.ImportacaoID == importacaoID {
			historicos = append(historicos, historico)
		}
	}

	return historicos, nil
}

type repoCategorias struct{ *memoria }

func (r repoCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return copiar(r.categorias[id]), nil
}

func copiarEstoque(estoque *domainestoque.EstoqueProduto) *domainestoque.EstoqueProduto {
	return &domainestoque.EstoqueProduto{
		Produto:      copiar(estoque.Produto),
		Saldos:       maps.Clone(estoque.Saldos),
		Lotes:        slices.Clone(estoque.Lotes),
		TotalAlocado: estoque.TotalAlocado,
	}
}

func (r repoImportacoes) aplicarOperacoes(
	operacoes []*domainestoque.OperacaoEstoque,
) (map[uuid.UUID]*domainestoque.EstoqueProduto, map[uuid.UUID]*domainestoque.Reserva, error) {
	estoques := make(map[uuid.UUID]*domainestoque.EstoqueProduto)
	reservas := make(map[uuid.UUID]*domainestoque.Reserva)
	for _, operacao := range operacoes {
		estoque, ok := estoques[operacao.ProdutoID]
		if !ok {
			original, existe := r.estoques[operacao.ProdutoID]
			if !existe {
				return nil, nil, domain.ErroValidacao("produto sem estoque")
			}
			estoque = copiarEstoque(original)
			estoques[operacao.ProdutoID] = estoque
		}
		if err := estoque.Aplicar(operacao); err != nil {
			return nil, nil, domain.ErroConflito("estoque do produto alterado por outra operação, recarregue e tente novamente")
		}
		if operacao.ReservaCriada != nil {
			reservas[operacao.ReservaCriada.ID] = copiar(operacao.ReservaCriada)
		}
	}

	return estoques, reservas, nil
}

func (r repoImportacoes) ItensDePedido(_ context.Context, ids []uuid.UUID, exceto uuid.UUID) ([]*domainimportacao.PedidoItemAlocavel, error) {
	itens := make([]*domainimportacao.PedidoItemAlocavel, 0, len(ids))
	for _, id := range ids {
		item, ok := r.pedidoItens[id]
		if !ok {
			continue
		}
		copia := copiar(item)
		copia.Reservado = 0
		for _, reserva := range r.reservas {
			if reserva.OrigemID != nil && *reserva.OrigemID == id && reserva.Ativa() {
				copia.Reservado += reserva.Quantidade
			}
		}
		copia.AlocadoEmOutras = 0
		for _, importacao := range r.importacoes {
			if importacao.ID == exceto || importacao.Status == domainimportacao.StatusCancelada || importacao.Status == domainimportacao.StatusRecebida {
				continue
			}
			for _, alocacao := range importacao.Alocacoes {
				if alocacao.PedidoItemID == id {
					copia.AlocadoEmOutras += alocacao.Quantidade
				}
			}
		}
		itens = append(itens, copia)
	}

	return itens, nil
}

type repoEstoque struct{ *memoria }

func (r repoEstoque) Carregar(_ context.Context, produtoID uuid.UUID, _ []uuid.UUID) (*domainestoque.EstoqueProduto, error) {
	estoque, ok := r.estoques[produtoID]
	if !ok {
		return nil, nil
	}

	return copiarEstoque(estoque), nil
}

type repoLocais struct{ *memoria }

func (r repoLocais) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Local, error) {
	return copiar(r.locais[id]), nil
}
