package estoque

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type EstoqueProduto struct {
	Produto      *Produto
	Saldos       map[uuid.UUID]SaldoLocal
	Lotes        []*Lote
	TotalAlocado int
}

type MovimentoLocalInput struct {
	Tipo         TipoMovimento
	Local        *Local
	Quantidade   Quantidade
	Custo        *Custo
	Reserva      *Reserva
	UsuarioID    uuid.UUID
	DocumentoRef *string
	Observacao   string
	Agora        time.Time
}

type TransferenciaInput struct {
	Origem       *Local
	Destino      *Local
	Quantidade   Quantidade
	UsuarioID    uuid.UUID
	DocumentoRef *string
	Observacao   string
	Agora        time.Time
}

type OperacaoEstoque struct {
	ProdutoID        uuid.UUID
	VersaoLida       int64
	Saldo            Saldo
	Reservado        Saldo
	Movimento        *Movimento
	Saldos           []SaldoLocal
	LotesCriados     []*Lote
	ReservaCriada    *Reserva
	ReservaEncerrada *Reserva
	AtualizadoEm     time.Time
}

func (e *EstoqueProduto) NaoAlocado() int {
	return e.Produto.Saldo.Quantidade() - e.TotalAlocado
}

func (e *EstoqueProduto) Aplicar(operacao *OperacaoEstoque) error {
	switch {
	case operacao == nil || operacao.ProdutoID != e.Produto.ID:
		return domain.ErroValidacao("operação de outro produto")
	case operacao.VersaoLida != e.Produto.VersaoEstoque:
		return domain.ErroConflito("operação lida de outra versão do estoque")
	}

	if e.Saldos == nil {
		e.Saldos = map[uuid.UUID]SaldoLocal{}
	}
	for _, saldo := range operacao.Saldos {
		e.TotalAlocado += saldo.Quantidade - e.SaldoNoLocal(saldo.LocalID).Quantidade
		e.Saldos[saldo.LocalID] = saldo
	}

	if operacao.Movimento != nil {
		e.Lotes = consumirLotes(e.Lotes, operacao.Movimento.Consumos)
	}
	e.Lotes = append(e.Lotes, operacao.LotesCriados...)

	e.Produto.Saldo = operacao.Saldo
	e.Produto.Reservado = operacao.Reservado
	e.Produto.VersaoEstoque++

	return nil
}

func consumirLotes(lotes []*Lote, consumos []ConsumoLote) []*Lote {
	if len(consumos) == 0 {
		return lotes
	}

	consumidoPorLote := make(map[uuid.UUID]int, len(consumos))
	for _, consumo := range consumos {
		consumidoPorLote[consumo.LoteID] += consumo.Quantidade
	}

	atualizados := make([]*Lote, 0, len(lotes))
	for _, lote := range lotes {
		consumido, ok := consumidoPorLote[lote.ID]
		if !ok {
			atualizados = append(atualizados, lote)
			continue
		}

		copia := *lote
		copia.QuantidadeAtual -= consumido
		atualizados = append(atualizados, &copia)
	}

	return atualizados
}

func (e *EstoqueProduto) SaldoNoLocal(localID uuid.UUID) SaldoLocal {
	if saldo, ok := e.Saldos[localID]; ok {
		return saldo
	}

	return SaldoLocal{ProdutoID: e.Produto.ID, LocalID: localID}
}

func (e *EstoqueProduto) Movimentar(input MovimentoLocalInput) (*OperacaoEstoque, error) {
	if err := validarMovimentoLocal(input); err != nil {
		return nil, err
	}

	if err := e.validarReservaDaSaida(input); err != nil {
		return nil, err
	}

	efeitoNoLocal := input.Tipo.EfeitoNoLocal(input.Quantidade)
	if err := e.validarEfeito(input, efeitoNoLocal); err != nil {
		return nil, err
	}

	saldoTotal, err := e.Produto.Saldo.Aplicar(input.Tipo.Efeito(input.Quantidade))
	if err != nil {
		return nil, err
	}

	saldoLocal := e.SaldoNoLocal(input.Local.ID)
	reservado := e.Produto.Reservado
	var baixada *Reserva
	if input.Reserva != nil {
		if baixada, err = input.Reserva.encerrar(StatusReservaBaixada, input.Agora); err != nil {
			return nil, err
		}
		if saldoLocal, reservado, err = e.devolverReservado(input.Reserva, input.Agora); err != nil {
			return nil, err
		}
	}
	saldoLocal.Quantidade += efeitoNoLocal
	saldoLocal.AtualizadoEm = input.Agora

	movimento, err := NovoMovimento(
		e.Produto.ID, input.Tipo, input.Quantidade, saldoTotal,
		input.UsuarioID, input.DocumentoRef, input.Observacao, input.Agora,
	)
	if err != nil {
		return nil, err
	}

	saldoLocalApos := SaldoDe(saldoLocal.Quantidade)
	movimento.LocalID = &input.Local.ID
	movimento.SaldoLocalApos = &saldoLocalApos
	movimento.Custo = input.Custo
	if baixada != nil {
		movimento.ReservaID = &baixada.ID
	}

	operacao := e.novaOperacao(saldoTotal, movimento, saldoLocal)
	operacao.Reservado = reservado
	operacao.ReservaEncerrada = baixada

	if efeitoNoLocal < 0 {
		consumos, err := consumirFIFO(e.lotesNoLocal(input.Local.ID), -efeitoNoLocal)
		if err != nil {
			return nil, err
		}
		movimento.Consumos = consumos

		return operacao, nil
	}

	operacao.LotesCriados = []*Lote{
		novoLote(e.Produto.ID, input.Local.ID, movimento.ID, nil, input.Custo, efeitoNoLocal, input.Agora, input.Agora),
	}

	return operacao, nil
}

func (e *EstoqueProduto) Transferir(input TransferenciaInput) (*OperacaoEstoque, error) {
	if err := validarTransferencia(input); err != nil {
		return nil, err
	}

	quantidade := input.Quantidade.Valor()
	origem := e.SaldoNoLocal(input.Origem.ID)
	if quantidade > origem.Disponivel() {
		return nil, domain.ErroValidacao("saldo disponível insuficiente no local de origem")
	}

	lotesDaOrigem := e.lotesNoLocal(input.Origem.ID)
	consumos, err := consumirFIFO(lotesDaOrigem, quantidade)
	if err != nil {
		return nil, err
	}

	origem.Quantidade -= quantidade
	origem.AtualizadoEm = input.Agora
	destino := e.SaldoNoLocal(input.Destino.ID)
	destino.Quantidade += quantidade
	destino.AtualizadoEm = input.Agora

	movimento, err := NovoMovimento(
		e.Produto.ID, TipoMovimentoTransferencia, input.Quantidade, e.Produto.Saldo,
		input.UsuarioID, input.DocumentoRef, input.Observacao, input.Agora,
	)
	if err != nil {
		return nil, err
	}

	saldoOrigemApos := SaldoDe(origem.Quantidade)
	movimento.LocalID = &input.Origem.ID
	movimento.LocalDestinoID = &input.Destino.ID
	movimento.SaldoLocalApos = &saldoOrigemApos
	movimento.Consumos = consumos

	operacao := e.novaOperacao(e.Produto.Saldo, movimento, origem, destino)

	entradas := make(map[uuid.UUID]time.Time, len(lotesDaOrigem))
	for _, lote := range lotesDaOrigem {
		entradas[lote.ID] = lote.EntradaEm
	}
	for _, consumo := range consumos {
		loteOrigemID := consumo.LoteID
		operacao.LotesCriados = append(operacao.LotesCriados, novoLote(
			e.Produto.ID, input.Destino.ID, movimento.ID, &loteOrigemID, consumo.Custo,
			consumo.Quantidade, entradas[consumo.LoteID], input.Agora,
		))
	}

	return operacao, nil
}

func (e *EstoqueProduto) novaOperacao(saldoTotal Saldo, movimento *Movimento, saldos ...SaldoLocal) *OperacaoEstoque {
	return &OperacaoEstoque{
		ProdutoID:    e.Produto.ID,
		VersaoLida:   e.Produto.VersaoEstoque,
		Saldo:        saldoTotal,
		Reservado:    e.Produto.Reservado,
		Movimento:    movimento,
		Saldos:       saldos,
		AtualizadoEm: movimento.CriadoEm,
	}
}

func (e *EstoqueProduto) validarReservaDaSaida(input MovimentoLocalInput) error {
	reserva := input.Reserva
	switch {
	case reserva == nil:
		return nil
	case input.Tipo != TipoMovimentoSaida:
		return domain.ErroValidacao("só a saída baixa reserva")
	case reserva.ProdutoID != e.Produto.ID || reserva.LocalID != input.Local.ID:
		return domain.ErroValidacao("reserva de outro produto ou local")
	case input.Quantidade.Absoluta() != reserva.Quantidade:
		return domain.ErroValidacao("saída que baixa reserva deve ter a quantidade da reserva")
	case !reserva.Ativa():
		return domain.ErroConflito("reserva já encerrada")
	}

	return nil
}

func (e *EstoqueProduto) lotesNoLocal(localID uuid.UUID) []*Lote {
	lotes := make([]*Lote, 0, len(e.Lotes))
	for _, lote := range e.Lotes {
		if lote.LocalID == localID && lote.QuantidadeAtual > 0 {
			lotes = append(lotes, lote)
		}
	}

	return lotes
}

func (e *EstoqueProduto) validarEfeito(input MovimentoLocalInput, efeitoNoLocal int) error {
	if efeitoNoLocal > 0 {
		if !input.Local.Ativo {
			return domain.ErroValidacao("local inativo não recebe entrada de estoque")
		}
		if input.Tipo == TipoMovimentoEntrada && input.Custo == nil {
			return domain.ErroValidacao("custo unitário é obrigatório na entrada")
		}
		if input.Tipo == TipoMovimentoImplantacao && efeitoNoLocal > e.NaoAlocado() {
			return domain.ErroValidacao("quantidade maior que o saldo ainda não alocado do produto")
		}

		return nil
	}

	if input.Custo != nil {
		return domain.ErroValidacao("custo unitário só é informado em movimento que cria lote")
	}
	disponivel := e.SaldoNoLocal(input.Local.ID).Disponivel()
	if input.Reserva != nil {
		disponivel += input.Reserva.Quantidade
	}
	if -efeitoNoLocal > disponivel {
		return domain.ErroValidacao("saldo disponível insuficiente no local")
	}

	return nil
}

func validarMovimentoLocal(input MovimentoLocalInput) error {
	if err := input.Tipo.Validado(); err != nil {
		return err
	}
	if input.Tipo == TipoMovimentoTransferencia {
		return domain.ErroValidacao("transferência tem operação própria")
	}
	if input.Local == nil {
		return domain.ErroValidacao("local do movimento é obrigatório")
	}
	if input.UsuarioID == uuid.Nil {
		return domain.ErroValidacao("usuário da movimentação não informado")
	}
	if input.Quantidade.Valor() == 0 {
		return domain.ErroValidacao("quantidade da movimentação deve ser diferente de zero")
	}

	return nil
}

func validarTransferencia(input TransferenciaInput) error {
	switch {
	case input.Origem == nil || input.Destino == nil:
		return domain.ErroValidacao("locais de origem e destino são obrigatórios")
	case input.Origem.ID == input.Destino.ID:
		return domain.ErroValidacao("local de destino deve ser diferente do de origem")
	case input.Origem.EmpresaID != input.Destino.EmpresaID:
		return domain.ErroValidacao("transferência só entre locais da mesma empresa")
	case !input.Destino.Ativo:
		return domain.ErroValidacao("local inativo não recebe entrada de estoque")
	case input.UsuarioID == uuid.Nil:
		return domain.ErroValidacao("usuário da movimentação não informado")
	case input.Quantidade.Valor() <= 0:
		return domain.ErroValidacao("quantidade da transferência deve ser maior que zero")
	}

	return nil
}
