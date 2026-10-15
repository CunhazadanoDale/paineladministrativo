package estoque

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoObservacaoReserva = 1000

type StatusReserva string

const (
	StatusReservaAtiva    StatusReserva = "ativa"
	StatusReservaBaixada  StatusReserva = "baixada"
	StatusReservaLiberada StatusReserva = "liberada"
)

func NovoStatusReserva(valor string) (StatusReserva, error) {
	status := StatusReserva(strings.ToLower(strings.TrimSpace(valor)))

	switch status {
	case StatusReservaAtiva, StatusReservaBaixada, StatusReservaLiberada:
		return status, nil
	default:
		return "", domain.ErroValidacao("status da reserva deve ser ativa, baixada ou liberada")
	}
}

func StatusReservaDe(valor string) StatusReserva {
	return StatusReserva(valor)
}

func (s StatusReserva) String() string {
	return string(s)
}

type OrigemReserva string

const (
	OrigemReservaManual     OrigemReserva = "manual"
	OrigemReservaPedidoItem OrigemReserva = "pedido_item"
)

func OrigemReservaDe(valor string) OrigemReserva {
	return OrigemReserva(valor)
}

func (o OrigemReserva) String() string {
	return string(o)
}

type Reserva struct {
	ID          uuid.UUID
	ProdutoID   uuid.UUID
	LocalID     uuid.UUID
	Quantidade  int
	Origem      OrigemReserva
	OrigemID    *uuid.UUID
	Status      StatusReserva
	UsuarioID   uuid.UUID
	Observacao  string
	CriadoEm    time.Time
	EncerradaEm *time.Time
}

func (r *Reserva) Ativa() bool {
	return r.Status == StatusReservaAtiva
}

func (r *Reserva) encerrar(status StatusReserva, agora time.Time) (*Reserva, error) {
	if !r.Ativa() {
		return nil, domain.ErroConflito("reserva já encerrada")
	}

	encerrada := *r
	encerrada.Status = status
	encerrada.EncerradaEm = &agora

	return &encerrada, nil
}

type ReservaInput struct {
	Local      *Local
	Quantidade Quantidade
	Origem     OrigemReserva
	OrigemID   *uuid.UUID
	UsuarioID  uuid.UUID
	Observacao string
	Agora      time.Time
}

func (e *EstoqueProduto) Reservar(input ReservaInput) (*OperacaoEstoque, error) {
	observacao, err := validarReserva(input)
	if err != nil {
		return nil, err
	}

	quantidade := input.Quantidade.Valor()
	saldoLocal := e.SaldoNoLocal(input.Local.ID)
	if quantidade > saldoLocal.Disponivel() {
		return nil, domain.ErroValidacao("saldo disponível insuficiente no local para a reserva")
	}

	reservado, err := e.Produto.Reservado.Aplicar(quantidade)
	if err != nil {
		return nil, err
	}

	saldoLocal.Reservado += quantidade
	saldoLocal.AtualizadoEm = input.Agora

	origem := input.Origem
	if origem == "" {
		origem = OrigemReservaManual
	}
	if err := validarOrigemReserva(origem, input.OrigemID); err != nil {
		return nil, err
	}

	operacao := e.operacaoDeReserva(reservado, input.Agora, saldoLocal)
	operacao.ReservaCriada = &Reserva{
		ID:         uuid.New(),
		ProdutoID:  e.Produto.ID,
		LocalID:    input.Local.ID,
		Quantidade: quantidade,
		Origem:     origem,
		OrigemID:   input.OrigemID,
		Status:     StatusReservaAtiva,
		UsuarioID:  input.UsuarioID,
		Observacao: observacao,
		CriadoEm:   input.Agora,
	}

	return operacao, nil
}

func (e *EstoqueProduto) LiberarReserva(reserva *Reserva, agora time.Time) (*OperacaoEstoque, error) {
	if reserva == nil || reserva.ProdutoID != e.Produto.ID {
		return nil, domain.ErroValidacao("reserva não pertence ao produto")
	}

	liberada, err := reserva.encerrar(StatusReservaLiberada, agora)
	if err != nil {
		return nil, err
	}

	saldoLocal, reservado, err := e.devolverReservado(reserva, agora)
	if err != nil {
		return nil, err
	}

	operacao := e.operacaoDeReserva(reservado, agora, saldoLocal)
	operacao.ReservaEncerrada = liberada

	return operacao, nil
}

func (e *EstoqueProduto) devolverReservado(reserva *Reserva, agora time.Time) (SaldoLocal, Saldo, error) {
	saldoLocal := e.SaldoNoLocal(reserva.LocalID)
	if saldoLocal.Reservado < reserva.Quantidade {
		return SaldoLocal{}, Saldo{}, domain.ErroConflito("reservado do local não cobre a reserva, recarregue e tente novamente")
	}

	reservado, err := e.Produto.Reservado.Aplicar(-reserva.Quantidade)
	if err != nil {
		return SaldoLocal{}, Saldo{}, domain.ErroConflito("reservado do produto não cobre a reserva, recarregue e tente novamente")
	}

	saldoLocal.Reservado -= reserva.Quantidade
	saldoLocal.AtualizadoEm = agora

	return saldoLocal, reservado, nil
}

func (e *EstoqueProduto) operacaoDeReserva(reservado Saldo, agora time.Time, saldo SaldoLocal) *OperacaoEstoque {
	return &OperacaoEstoque{
		ProdutoID:    e.Produto.ID,
		VersaoLida:   e.Produto.VersaoEstoque,
		Saldo:        e.Produto.Saldo,
		Reservado:    reservado,
		Saldos:       []SaldoLocal{saldo},
		AtualizadoEm: agora,
	}
}

func validarOrigemReserva(origem OrigemReserva, origemID *uuid.UUID) error {
	switch origem {
	case OrigemReservaManual:
		return nil
	case OrigemReservaPedidoItem:
		if origemID == nil || *origemID == uuid.Nil {
			return domain.ErroValidacao("reserva de item de pedido exige o item de origem")
		}

		return nil
	default:
		return domain.ErroValidacao("origem da reserva deve ser manual ou pedido_item")
	}
}

func validarReserva(input ReservaInput) (string, error) {
	switch {
	case input.Local == nil:
		return "", domain.ErroValidacao("local da reserva é obrigatório")
	case input.UsuarioID == uuid.Nil:
		return "", domain.ErroValidacao("usuário da reserva não informado")
	case input.Quantidade.Valor() <= 0:
		return "", domain.ErroValidacao("quantidade da reserva deve ser maior que zero")
	}

	observacao := strings.TrimSpace(input.Observacao)
	if utf8.RuneCountInString(observacao) > tamanhoMaximoObservacaoReserva {
		return "", domain.ErroValidacao("observação da reserva deve ter no máximo 1000 caracteres")
	}

	return observacao, nil
}
