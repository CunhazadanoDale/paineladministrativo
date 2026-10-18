package receber

import (
	"fmt"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoMotivoEstorno = 500

type TipoRecebimento string

const (
	TipoRecebimentoConta   TipoRecebimento = "conta"
	TipoRecebimentoFomento TipoRecebimento = "fomento"
)

func NovoTipoRecebimento(valor string) (TipoRecebimento, error) {
	tipo := TipoRecebimento(strings.ToLower(strings.TrimSpace(valor)))

	switch tipo {
	case TipoRecebimentoConta, TipoRecebimentoFomento:
		return tipo, nil
	case "":
		return TipoRecebimentoConta, nil
	default:
		return "", domain.ErroValidacao("tipo do recebimento deve ser conta ou fomento")
	}
}

func TipoRecebimentoDe(valor string) TipoRecebimento {
	return TipoRecebimento(valor)
}

func (t TipoRecebimento) String() string {
	return string(t)
}

type Recebimento struct {
	ID                     uuid.UUID
	TituloID               uuid.UUID
	Tipo                   TipoRecebimento
	ContaBancariaID        *uuid.UUID
	RecebidoEm             time.Time
	ValorCentavos          int64
	JurosCentavos          int64
	DescontoCentavos       int64
	StatusTituloAnterior   Status
	RegistrouEntradaPedido bool
	Estornado              bool
	Estorno                *EstornoRecebimento
	Repasse                *Repasse
	UsuarioID              uuid.UUID
	CriadoEm               time.Time
}

type EstornoRecebimento struct {
	ID            uuid.UUID
	RecebimentoID uuid.UUID
	Motivo        string
	EstornadoEm   time.Time
	UsuarioID     uuid.UUID
	CriadoEm      time.Time
}

type DadosRecebimento struct {
	Tipo             string
	ContaBancariaID  *uuid.UUID
	RecebidoEm       time.Time
	ValorCentavos    int64
	JurosCentavos    int64
	DescontoCentavos int64
	Repasse          *DadosRepasse
}

func (r *Recebimento) AbatidoCentavos() int64 {
	return r.ValorCentavos + r.DescontoCentavos
}

func (r *Recebimento) EntradaNoCaixaCentavos() int64 {
	if r.Tipo != TipoRecebimentoConta {
		return 0
	}

	return r.ValorCentavos + r.JurosCentavos
}

func (t *Titulo) PodeReceber() error {
	if !t.Status.EmAberto() {
		return domain.ErroConflito("só o título em carteira, em banco ou antecipado recebe baixa")
	}

	return nil
}

func (t *Titulo) Receber(dados DadosRecebimento, usuarioID uuid.UUID, agora time.Time) (*Recebimento, *Historico, error) {
	if err := t.PodeReceber(); err != nil {
		return nil, nil, err
	}

	tipo, err := NovoTipoRecebimento(dados.Tipo)
	if err != nil {
		return nil, nil, err
	}
	conta := idOpcional(dados.ContaBancariaID)
	switch {
	case tipo == TipoRecebimentoConta && conta == nil:
		return nil, nil, domain.ErroValidacao("conta bancária do recebimento é obrigatória")
	case tipo == TipoRecebimentoFomento && conta != nil:
		return nil, nil, domain.ErroValidacao("recebimento liquidado no fomento não tem conta bancária")
	case tipo == TipoRecebimentoFomento && t.Status != StatusAntecipado:
		return nil, nil, domain.ErroConflito("só o título antecipado é liquidado no fomento")
	case dados.Repasse != nil && (tipo != TipoRecebimentoConta || t.Status != StatusAntecipado):
		return nil, nil, domain.ErroValidacao("repasse ao fomento só vale para título antecipado recebido em conta própria")
	}

	recebidoEm, err := dataObrigatoria(dados.RecebidoEm, feminino("data do recebimento"))
	if err != nil {
		return nil, nil, err
	}
	if recebidoEm.After(Hoje(agora)) {
		return nil, nil, domain.ErroValidacao("data do recebimento não pode estar no futuro")
	}
	switch {
	case dados.ValorCentavos <= 0:
		return nil, nil, domain.ErroValidacao("valor do recebimento deve ser maior que zero")
	case dados.JurosCentavos < 0:
		return nil, nil, domain.ErroValidacao("juros do recebimento não podem ser negativos")
	case dados.DescontoCentavos < 0:
		return nil, nil, domain.ErroValidacao("desconto do recebimento não pode ser negativo")
	case dados.ValorCentavos+dados.DescontoCentavos > t.SaldoCentavos():
		return nil, nil, domain.ErroValidacao(fmt.Sprintf("valor mais desconto passa do saldo do título (%d centavos)", t.SaldoCentavos()))
	}

	anterior := t.Status
	recebimento := &Recebimento{
		ID:                   uuid.Must(uuid.NewV7()),
		TituloID:             t.ID,
		Tipo:                 tipo,
		ContaBancariaID:      conta,
		RecebidoEm:           recebidoEm,
		ValorCentavos:        dados.ValorCentavos,
		JurosCentavos:        dados.JurosCentavos,
		DescontoCentavos:     dados.DescontoCentavos,
		StatusTituloAnterior: anterior,
		UsuarioID:            usuarioID,
		CriadoEm:             agora,
	}
	if tipo == TipoRecebimentoConta && anterior == StatusAntecipado {
		if recebimento.Repasse, err = novoRepasse(recebimento, dados.Repasse, usuarioID, agora); err != nil {
			return nil, nil, err
		}
	}

	t.RecebidoCentavos += recebimento.AbatidoCentavos()
	if t.SaldoCentavos() == 0 {
		t.Status = StatusRecebido
	}
	t.AtualizadoEm = agora

	descricao := fmt.Sprintf("%s em %s: valor %d, juros %d, desconto %d centavos", tipo, recebidoEm.Format(formatoData),
		recebimento.ValorCentavos, recebimento.JurosCentavos, recebimento.DescontoCentavos)
	if recebimento.Repasse != nil {
		descricao += "; recebido em conta própria, gera repasse ao fomento"
	}

	return recebimento, NovoHistorico(t.ID, usuarioID, EventoRecebido, &anterior, t.Status, descricao, agora), nil
}

func (t *Titulo) EstornarRecebimento(
	recebimento *Recebimento, motivo string, estornadoEm time.Time, usuarioID uuid.UUID, agora time.Time,
) (*EstornoRecebimento, *Historico, error) {
	switch {
	case recebimento == nil || recebimento.TituloID != t.ID:
		return nil, nil, domain.ErroNaoEncontrado("recebimento não encontrado")
	case recebimento.Estornado:
		return nil, nil, domain.ErroConflito("o recebimento já foi estornado")
	case recebimento.RegistrouEntradaPedido:
		return nil, nil, domain.ErroConflito("o recebimento registrou a entrada paga do pedido e não pode ser estornado")
	case recebimento.Repasse != nil:
		return nil, nil, domain.ErroConflito("o recebimento gerou repasse ao fomento e não pode ser estornado; trate a conta a pagar do repasse")
	}

	motivo, err := textoObrigatorio(motivo, masculino("motivo do estorno"), tamanhoMaximoMotivoEstorno)
	if err != nil {
		return nil, nil, err
	}
	data, err := dataObrigatoria(estornadoEm, feminino("data do estorno"))
	if err != nil {
		return nil, nil, err
	}
	if data.After(Hoje(agora)) {
		return nil, nil, domain.ErroValidacao("data do estorno não pode estar no futuro")
	}
	if data.Before(recebimento.RecebidoEm) {
		return nil, nil, domain.ErroValidacao("data do estorno não pode ser anterior à do recebimento")
	}

	anterior := t.Status
	t.RecebidoCentavos -= recebimento.AbatidoCentavos()
	if t.Status == StatusRecebido {
		t.Status = recebimento.StatusTituloAnterior
	}
	t.AtualizadoEm = agora
	recebimento.Estornado = true

	estorno := &EstornoRecebimento{
		ID:            uuid.New(),
		RecebimentoID: recebimento.ID,
		Motivo:        motivo,
		EstornadoEm:   data,
		UsuarioID:     usuarioID,
		CriadoEm:      agora,
	}
	recebimento.Estorno = estorno

	return estorno, NovoHistorico(t.ID, usuarioID, EventoRecebimentoEstornado, &anterior, t.Status, motivo, agora), nil
}
