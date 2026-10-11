package estoque_test

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var (
	_ portsout.EmpresaRepository = (*repositorioEmpresas)(nil)
	_ portsout.LocalRepository   = (*repositorioLocais)(nil)
	_ portsout.EstoqueRepository = (*repositorioEstoque)(nil)
	_ portsout.LoteRepository    = (*repositorioLotes)(nil)
	_ portsout.ReservaRepository = (*repositorioReservas)(nil)
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

type repositorioLocais struct {
	itens   map[uuid.UUID]*domainestoque.Local
	estoque *repositorioEstoque
}

func novoRepositorioLocais() *repositorioLocais {
	return &repositorioLocais{itens: map[uuid.UUID]*domainestoque.Local{}}
}

func (r *repositorioLocais) Criar(_ context.Context, local *domainestoque.Local) (uuid.UUID, error) {
	for _, existente := range r.itens {
		if existente.EmpresaID == local.EmpresaID && strings.EqualFold(existente.Nome, local.Nome) {
			return uuid.Nil, domain.ErroConflito("já existe um registro com estes dados")
		}
	}

	copia := *local
	r.itens[local.ID] = &copia

	return local.ID, nil
}

func (r *repositorioLocais) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Local, error) {
	local, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *local
	return &copia, nil
}

func (r *repositorioLocais) ObterVarios(ctx context.Context, ids []uuid.UUID) ([]*domainestoque.Local, error) {
	itens := make([]*domainestoque.Local, 0, len(ids))
	for _, id := range ids {
		if local, _ := r.Obter(ctx, id); local != nil {
			itens = append(itens, local)
		}
	}

	return ordenarLocais(itens), nil
}

func (r *repositorioLocais) Listar(_ context.Context, filtro portsout.LocalFiltro) ([]*domainestoque.Local, error) {
	itens := make([]*domainestoque.Local, 0, len(r.itens))
	for _, local := range r.itens {
		if filtro.Empresa != nil && local.EmpresaID != *filtro.Empresa {
			continue
		}
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, local.EmpresaID) {
			continue
		}
		if filtro.Ativo != nil && local.Ativo != *filtro.Ativo {
			continue
		}
		copia := *local
		itens = append(itens, &copia)
	}

	return ordenarLocais(itens), nil
}

func (r *repositorioLocais) Atualizar(_ context.Context, local *domainestoque.Local) (bool, error) {
	if _, ok := r.itens[local.ID]; !ok {
		return false, nil
	}
	for _, existente := range r.itens {
		if existente.ID != local.ID && existente.EmpresaID == local.EmpresaID && strings.EqualFold(existente.Nome, local.Nome) {
			return false, domain.ErroConflito("já existe um registro com estes dados")
		}
	}

	copia := *local
	r.itens[local.ID] = &copia

	return true, nil
}

func (r *repositorioLocais) DesativarSemSaldo(_ context.Context, id uuid.UUID, agora time.Time) (bool, error) {
	local, ok := r.itens[id]
	if !ok {
		return false, nil
	}
	for _, saldos := range r.estoque.saldos {
		if saldo, ok := saldos[id]; ok && saldo.Quantidade > 0 {
			return false, nil
		}
	}

	local.Ativo = false
	local.AtualizadoEm = agora

	return true, nil
}

func (r *repositorioLocais) Remover(_ context.Context, id uuid.UUID) (bool, error) {
	if _, ok := r.itens[id]; !ok {
		return false, nil
	}
	for _, saldos := range r.estoque.saldos {
		if _, ok := saldos[id]; ok {
			return false, domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
		}
	}

	delete(r.itens, id)

	return true, nil
}

func ordenarLocais(itens []*domainestoque.Local) []*domainestoque.Local {
	slices.SortFunc(itens, func(a, b *domainestoque.Local) int {
		if comparacao := strings.Compare(a.Nome, b.Nome); comparacao != 0 {
			return comparacao
		}

		return bytes.Compare(a.ID[:], b.ID[:])
	})

	return itens
}

type repositorioEstoque struct {
	produtos *repositorioProdutos
	locais   *repositorioLocais
	saldos   map[uuid.UUID]map[uuid.UUID]domainestoque.SaldoLocal
	lotes    []*domainestoque.Lote
	reservas map[uuid.UUID]*domainestoque.Reserva
}

func novoRepositorioEstoque(produtos *repositorioProdutos, locais *repositorioLocais) *repositorioEstoque {
	estoque := &repositorioEstoque{
		produtos: produtos,
		locais:   locais,
		saldos:   map[uuid.UUID]map[uuid.UUID]domainestoque.SaldoLocal{},
		reservas: map[uuid.UUID]*domainestoque.Reserva{},
	}
	locais.estoque = estoque

	return estoque
}

func (r *repositorioEstoque) Carregar(_ context.Context, produtoID uuid.UUID, localIDs []uuid.UUID) (*domainestoque.EstoqueProduto, error) {
	produto, ok := r.produtos.itens[produtoID]
	if !ok {
		return nil, nil
	}

	copiaProduto := *produto
	estoque := &domainestoque.EstoqueProduto{
		Produto: &copiaProduto,
		Saldos:  map[uuid.UUID]domainestoque.SaldoLocal{},
	}
	for localID, saldo := range r.saldos[produtoID] {
		estoque.TotalAlocado += saldo.Quantidade
		if slices.Contains(localIDs, localID) {
			estoque.Saldos[localID] = saldo
		}
	}
	for _, lote := range r.lotes {
		if lote.ProdutoID == produtoID && lote.QuantidadeAtual > 0 && slices.Contains(localIDs, lote.LocalID) {
			copia := *lote
			estoque.Lotes = append(estoque.Lotes, &copia)
		}
	}

	return estoque, nil
}

func (r *repositorioEstoque) Gravar(_ context.Context, operacao *domainestoque.OperacaoEstoque) error {
	produto, ok := r.produtos.itens[operacao.ProdutoID]
	if !ok {
		return domain.ErroNaoEncontrado("produto não encontrado")
	}
	if produto.VersaoEstoque != operacao.VersaoLida {
		return domain.ErroConflito("estoque do produto alterado por outra operação, recarregue e tente novamente")
	}
	for _, lote := range operacao.LotesCriados {
		if local, ok := r.locais.itens[lote.LocalID]; !ok || !local.Ativo {
			return domain.ErroValidacao("local inativo não recebe entrada de estoque")
		}
	}
	consumos := []domainestoque.ConsumoLote{}
	if operacao.Movimento != nil {
		consumos = operacao.Movimento.Consumos
	}
	for _, consumo := range consumos {
		if lote := r.lote(consumo.LoteID); lote == nil || lote.QuantidadeAtual < consumo.Quantidade {
			return domain.ErroConflito("lote consumido por outra operação, recarregue e tente novamente")
		}
	}
	if encerrada := operacao.ReservaEncerrada; encerrada != nil {
		if salva, ok := r.reservas[encerrada.ID]; !ok || !salva.Ativa() {
			return domain.ErroConflito("reserva já encerrada")
		}
	}

	produto.Saldo = operacao.Saldo
	produto.Reservado = operacao.Reservado
	produto.VersaoEstoque++

	if r.saldos[operacao.ProdutoID] == nil {
		r.saldos[operacao.ProdutoID] = map[uuid.UUID]domainestoque.SaldoLocal{}
	}
	for _, saldo := range operacao.Saldos {
		r.saldos[operacao.ProdutoID][saldo.LocalID] = saldo
	}
	for _, consumo := range consumos {
		r.lote(consumo.LoteID).QuantidadeAtual -= consumo.Quantidade
	}
	for _, lote := range operacao.LotesCriados {
		copia := *lote
		r.lotes = append(r.lotes, &copia)
	}
	for _, reserva := range []*domainestoque.Reserva{operacao.ReservaCriada, operacao.ReservaEncerrada} {
		if reserva != nil {
			copia := *reserva
			r.reservas[reserva.ID] = &copia
		}
	}

	if operacao.Movimento != nil {
		movimento := *operacao.Movimento
		r.produtos.movimentos[operacao.ProdutoID] = append(r.produtos.movimentos[operacao.ProdutoID], &movimento)
	}

	return nil
}

type repositorioReservas struct {
	estoque *repositorioEstoque
}

func (r *repositorioReservas) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Reserva, error) {
	reserva, ok := r.estoque.reservas[id]
	if !ok {
		return nil, nil
	}

	copia := *reserva
	return &copia, nil
}

func (r *repositorioReservas) Listar(_ context.Context, filtro portsout.ReservaFiltro) ([]*domainestoque.Reserva, error) {
	itens := make([]*domainestoque.Reserva, 0)
	for _, reserva := range r.estoque.reservas {
		if reserva.ProdutoID != filtro.ProdutoID {
			continue
		}
		if filtro.LocalID != nil && reserva.LocalID != *filtro.LocalID {
			continue
		}
		if filtro.Status != "" && reserva.Status != filtro.Status {
			continue
		}
		if filtro.EmpresaIDs != nil {
			local, ok := r.estoque.locais.itens[reserva.LocalID]
			if !ok || !slices.Contains(filtro.EmpresaIDs, local.EmpresaID) {
				continue
			}
		}
		copia := *reserva
		itens = append(itens, &copia)
	}

	slices.SortFunc(itens, func(a, b *domainestoque.Reserva) int {
		return b.CriadoEm.Compare(a.CriadoEm)
	})

	return itens, nil
}

func (r *repositorioEstoque) lote(id uuid.UUID) *domainestoque.Lote {
	for _, lote := range r.lotes {
		if lote.ID == id {
			return lote
		}
	}

	return nil
}

func (r *repositorioEstoque) SaldosDoProduto(_ context.Context, produtoID uuid.UUID, empresaIDs []uuid.UUID) ([]*domainestoque.SaldoLocalDetalhe, error) {
	itens := make([]*domainestoque.SaldoLocalDetalhe, 0)
	for _, saldo := range r.saldos[produtoID] {
		detalhe := r.detalhar(saldo)
		if empresaIDs != nil && !slices.Contains(empresaIDs, detalhe.EmpresaID) {
			continue
		}
		itens = append(itens, detalhe)
	}

	slices.SortFunc(itens, func(a, b *domainestoque.SaldoLocalDetalhe) int {
		return strings.Compare(a.LocalNome, b.LocalNome)
	})

	return itens, nil
}

func (r *repositorioEstoque) SaldosDoLocal(_ context.Context, filtro portsout.SaldoDoLocalFiltro) ([]*domainestoque.SaldoLocalDetalhe, error) {
	itens := make([]*domainestoque.SaldoLocalDetalhe, 0)
	for _, saldos := range r.saldos {
		saldo, ok := saldos[filtro.LocalID]
		if !ok || (filtro.ComSaldo && saldo.Quantidade == 0) {
			continue
		}
		itens = append(itens, r.detalhar(saldo))
	}

	slices.SortFunc(itens, func(a, b *domainestoque.SaldoLocalDetalhe) int {
		return strings.Compare(a.ProdutoNome, b.ProdutoNome)
	})

	return itens, nil
}

func (r *repositorioEstoque) TotalAlocado(_ context.Context, produtoID uuid.UUID) (int, error) {
	total := 0
	for _, saldo := range r.saldos[produtoID] {
		total += saldo.Quantidade
	}

	return total, nil
}

func (r *repositorioEstoque) detalhar(saldo domainestoque.SaldoLocal) *domainestoque.SaldoLocalDetalhe {
	detalhe := &domainestoque.SaldoLocalDetalhe{SaldoLocal: saldo}
	if local, ok := r.locais.itens[saldo.LocalID]; ok {
		detalhe.LocalNome = local.Nome
		detalhe.EmpresaID = local.EmpresaID
	}
	if produto, ok := r.produtos.itens[saldo.ProdutoID]; ok {
		detalhe.ProdutoNome = produto.Nome
		detalhe.ProdutoCodigo = produto.Codigo
		detalhe.UnidadeMedida = produto.UnidadeMedida
	}

	return detalhe
}

type repositorioLotes struct {
	estoque *repositorioEstoque
}

func (r *repositorioLotes) Listar(_ context.Context, filtro portsout.LoteFiltro) ([]*domainestoque.Lote, error) {
	itens := make([]*domainestoque.Lote, 0)
	for _, lote := range r.estoque.lotes {
		if lote.ProdutoID != filtro.ProdutoID {
			continue
		}
		if filtro.LocalID != nil && lote.LocalID != *filtro.LocalID {
			continue
		}
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, r.empresaDoLocal(lote.LocalID)) {
			continue
		}
		if filtro.ComSaldo && lote.QuantidadeAtual == 0 {
			continue
		}
		copia := *lote
		itens = append(itens, &copia)
	}

	slices.SortFunc(itens, func(a, b *domainestoque.Lote) int {
		if comparacao := a.EntradaEm.Compare(b.EntradaEm); comparacao != 0 {
			return comparacao
		}

		return bytes.Compare(a.ID[:], b.ID[:])
	})

	return itens, nil
}

func (r *repositorioLotes) empresaDoLocal(localID uuid.UUID) uuid.UUID {
	if local, ok := r.estoque.locais.itens[localID]; ok {
		return local.EmpresaID
	}

	return uuid.Nil
}

func movimentoVisivelPara(locais *repositorioLocais, movimento *domainestoque.Movimento, filtro portsout.MovimentoFiltro) bool {
	if filtro.LocalID != nil {
		naOrigem := movimento.LocalID != nil && *movimento.LocalID == *filtro.LocalID
		noDestino := movimento.LocalDestinoID != nil && *movimento.LocalDestinoID == *filtro.LocalID
		if !naOrigem && !noDestino {
			return false
		}
	}

	return movimentoDasEmpresas(locais, movimento, filtro.EmpresaIDs)
}

func movimentoDasEmpresas(locais *repositorioLocais, movimento *domainestoque.Movimento, empresaIDs []uuid.UUID) bool {
	if empresaIDs == nil || movimento.LocalID == nil {
		return true
	}

	local, ok := locais.itens[*movimento.LocalID]

	return ok && slices.Contains(empresaIDs, local.EmpresaID)
}
