package caixa

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	JanelaDeConciliacaoDias          = 3
	MaximoDeMovimentosPorConciliacao = 50
	tamanhoMaximoMotivo              = 500
)

type ModoConciliacao string

const (
	ConciliacaoManual     ModoConciliacao = "manual"
	ConciliacaoAutomatica ModoConciliacao = "automatica"
)

func ModoConciliacaoDe(valor string) ModoConciliacao {
	return ModoConciliacao(valor)
}

func (m ModoConciliacao) String() string {
	return string(m)
}

type SituacaoLancamento string

const (
	LancamentoPendente   SituacaoLancamento = "pendente"
	LancamentoConciliado SituacaoLancamento = "conciliado"
	LancamentoInvalidado SituacaoLancamento = "invalidado"
)

func NovaSituacaoLancamento(valor string) (SituacaoLancamento, error) {
	situacao := SituacaoLancamento(strings.ToLower(strings.TrimSpace(valor)))
	switch situacao {
	case LancamentoPendente, LancamentoConciliado, LancamentoInvalidado:
		return situacao, nil
	default:
		return "", domain.ErroValidacao("situação deve ser pendente, conciliado ou invalidado")
	}
}

func SituacaoLancamentoDe(valor string) SituacaoLancamento {
	return SituacaoLancamento(valor)
}

func (s SituacaoLancamento) String() string {
	return string(s)
}

type ItemConciliacao struct {
	Tipo          TipoMovimento
	AlvoID        uuid.UUID
	ValorCentavos int64
	Estornado     bool
}

type Conciliacao struct {
	ID                  uuid.UUID
	ExtratoLancamentoID uuid.UUID
	ContaBancariaID     uuid.UUID
	Modo                ModoConciliacao
	Vigente             bool
	Itens               []ItemConciliacao
	UsuarioID           uuid.UUID
	CriadoEm            time.Time
	DesfeitaEm          *time.Time
	DesfeitaPor         *uuid.UUID
	MotivoDesfazer      *string
}

type DadosConciliacao struct {
	Lancamento *LancamentoExtrato
	Movimentos []*MovimentoSistema
	Modo       ModoConciliacao
	UsuarioID  uuid.UUID
	Agora      time.Time
}

func NovaConciliacao(dados DadosConciliacao) (*Conciliacao, error) {
	lancamento := dados.Lancamento
	if lancamento == nil {
		return nil, domain.ErroValidacao("lançamento do extrato não informado")
	}
	if dados.UsuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}
	if dados.Modo != ConciliacaoManual && dados.Modo != ConciliacaoAutomatica {
		return nil, domain.ErroValidacao("modo de conciliação inválido")
	}
	if lancamento.ConciliacaoID != nil {
		return nil, ErroLancamentoJaConciliado()
	}
	if len(dados.Movimentos) == 0 {
		return nil, domain.ErroValidacao("informe ao menos um movimento para conciliar")
	}
	if len(dados.Movimentos) > MaximoDeMovimentosPorConciliacao {
		return nil, domain.ErroValidacao("uma conciliação aceita no máximo " + strconv.Itoa(MaximoDeMovimentosPorConciliacao) + " movimentos")
	}

	vistos := make(map[ReferenciaMovimento]struct{}, len(dados.Movimentos))
	itens := make([]ItemConciliacao, 0, len(dados.Movimentos))
	var soma int64
	for _, movimento := range dados.Movimentos {
		if err := conferirMovimento(lancamento, movimento); err != nil {
			return nil, err
		}
		if _, repetido := vistos[movimento.Referencia()]; repetido {
			return nil, domain.ErroValidacao("movimento repetido na conciliação")
		}
		vistos[movimento.Referencia()] = struct{}{}

		itens = append(itens, ItemConciliacao{Tipo: movimento.Tipo, AlvoID: movimento.ID, ValorCentavos: movimento.ValorCentavos})
		soma += movimento.ValorCentavos
	}

	if soma != lancamento.ValorCentavos {
		return nil, domain.ErroValidacao("a soma dos movimentos (" + FormatarCentavos(soma) +
			") é diferente do valor do lançamento (" + FormatarCentavos(lancamento.ValorCentavos) + ")")
	}

	return &Conciliacao{
		ID:                  uuid.New(),
		ExtratoLancamentoID: lancamento.ID,
		ContaBancariaID:     lancamento.ContaBancariaID,
		Modo:                dados.Modo,
		Vigente:             true,
		Itens:               itens,
		UsuarioID:           dados.UsuarioID,
		CriadoEm:            dados.Agora,
	}, nil
}

func (c *Conciliacao) Invalidada() bool {
	if !c.Vigente {
		return false
	}
	for _, item := range c.Itens {
		if item.Estornado {
			return true
		}
	}

	return false
}

func (c *Conciliacao) Desfazer(usuarioID uuid.UUID, motivo string, agora time.Time) error {
	if !c.Vigente {
		return ErroConciliacaoJaDesfeita()
	}
	if usuarioID == uuid.Nil {
		return domain.ErroValidacao("usuário não informado")
	}

	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return domain.ErroValidacao("motivo para desfazer a conciliação é obrigatório")
	}
	if utf8.RuneCountInString(motivo) > tamanhoMaximoMotivo {
		return domain.ErroValidacao("motivo deve ter no máximo " + strconv.Itoa(tamanhoMaximoMotivo) + " caracteres")
	}

	c.Vigente = false
	c.DesfeitaEm = &agora
	c.DesfeitaPor = &usuarioID
	c.MotivoDesfazer = &motivo

	return nil
}

func conferirMovimento(lancamento *LancamentoExtrato, movimento *MovimentoSistema) error {
	if movimento == nil {
		return domain.ErroValidacao("movimento não informado")
	}
	if movimento.ContaBancariaID != lancamento.ContaBancariaID {
		return domain.ErroValidacao("o movimento é de outra conta bancária")
	}
	if movimento.Estornado {
		return domain.ErroValidacao("movimento estornado não pode ser conciliado")
	}
	if movimento.ConciliacaoID != nil {
		return ErroMovimentoJaConciliado()
	}
	if (movimento.ValorCentavos > 0) != (lancamento.ValorCentavos > 0) {
		return domain.ErroValidacao("movimento de entrada só concilia com crédito, e de saída só com débito")
	}
	if !DentroDaJanela(lancamento.Data, movimento.Data) {
		return domain.ErroValidacao("movimento fora da janela de " + strconv.Itoa(JanelaDeConciliacaoDias) + " dias do lançamento")
	}

	return nil
}

func DentroDaJanela(dataDoLancamento, dataDoMovimento time.Time) bool {
	diferenca := DataDe(dataDoMovimento).Sub(DataDe(dataDoLancamento))
	if diferenca < 0 {
		diferenca = -diferenca
	}

	return diferenca <= JanelaDeConciliacaoDias*24*time.Hour
}

func FormatarCentavos(centavos int64) string {
	sinal := ""
	if centavos < 0 {
		sinal = "-"
		centavos = -centavos
	}

	centavosTexto := strconv.FormatInt(centavos%100, 10)
	if len(centavosTexto) == 1 {
		centavosTexto = "0" + centavosTexto
	}

	return sinal + strconv.FormatInt(centavos/100, 10) + "," + centavosTexto
}
