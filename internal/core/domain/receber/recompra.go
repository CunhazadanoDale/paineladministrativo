package receber

import (
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoMotivoRecompra = 500

type Recompra struct {
	ID                    uuid.UUID
	TituloID              uuid.UUID
	AntecipacaoID         uuid.UUID
	RecompradoEm          time.Time
	EncargosCentavos      int64
	ValorCentavos         int64
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
	SolicitacaoID         uuid.UUID
	Motivo                string
	UsuarioID             uuid.UUID
	CriadoEm              time.Time
}

type DadosRecompra struct {
	RecompradoEm          time.Time
	EncargosCentavos      int64
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
	Motivo                string
}

type Repasse struct {
	ID                    uuid.UUID
	RecebimentoID         uuid.UUID
	AntecipacaoID         uuid.UUID
	ValorCentavos         int64
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
	SolicitacaoID         uuid.UUID
	UsuarioID             uuid.UUID
	CriadoEm              time.Time
}

type DadosRepasse struct {
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
}

func (t *Titulo) PodeSerRecomprado() error {
	if t.Status != StatusAntecipado {
		return domain.ErroConflito("só o título antecipado é recomprado")
	}

	return nil
}

func (t *Titulo) Recomprar(vinculo *AntecipacaoTitulo, dados DadosRecompra, usuarioID uuid.UUID, agora time.Time) (*Recompra, *Historico, error) {
	if err := t.PodeSerRecomprado(); err != nil {
		return nil, nil, err
	}
	if vinculo == nil || vinculo.TituloID != t.ID || !vinculo.Vigente {
		return nil, nil, domain.ErroConflito("o título não está numa antecipação vigente")
	}

	recompradoEm, err := dataObrigatoria(dados.RecompradoEm, feminino("data da recompra"))
	if err != nil {
		return nil, nil, err
	}
	if recompradoEm.After(Hoje(agora)) {
		return nil, nil, domain.ErroValidacao("data da recompra não pode estar no futuro")
	}
	vencimento, err := dataObrigatoria(dados.Vencimento, masculino("vencimento da recompra"))
	if err != nil {
		return nil, nil, err
	}
	if dados.EncargosCentavos < 0 {
		return nil, nil, domain.ErroValidacao("encargos da recompra não podem ser negativos")
	}
	if dados.CategoriaFinanceiraID == uuid.Nil {
		return nil, nil, domain.ErroValidacao("categoria financeira da recompra é obrigatória")
	}
	motivo, err := textoObrigatorio(dados.Motivo, masculino("motivo da recompra"), tamanhoMaximoMotivoRecompra)
	if err != nil {
		return nil, nil, err
	}

	recompra := &Recompra{
		ID:                    uuid.New(),
		TituloID:              t.ID,
		AntecipacaoID:         vinculo.AntecipacaoID,
		RecompradoEm:          recompradoEm,
		EncargosCentavos:      dados.EncargosCentavos,
		ValorCentavos:         t.SaldoCentavos() + dados.EncargosCentavos,
		Vencimento:            vencimento,
		CategoriaFinanceiraID: dados.CategoriaFinanceiraID,
		Motivo:                motivo,
		UsuarioID:             usuarioID,
		CriadoEm:              agora,
	}

	anterior := t.Status
	t.Status = StatusCarteira
	t.ContaBancariaID = nil
	t.AtualizadoEm = agora
	vinculo.Vigente = false

	descricao := fmt.Sprintf("recompra de %d centavos (encargos %d): %s", recompra.ValorCentavos, recompra.EncargosCentavos, motivo)

	return recompra, NovoHistorico(t.ID, usuarioID, EventoRecomprado, &anterior, t.Status, descricao, agora), nil
}

func (r *Recompra) Observacao(titulo *Titulo) string {
	return fmt.Sprintf("recompra do título %s/%d: %s", titulo.Numero, titulo.Parcela, r.Motivo)
}

func novoRepasse(recebimento *Recebimento, dados *DadosRepasse, usuarioID uuid.UUID, agora time.Time) (*Repasse, error) {
	if dados == nil {
		return nil, domain.ErroValidacao("título antecipado recebido em conta própria exige o repasse ao fomento (vencimento e categoria)")
	}
	vencimento, err := dataObrigatoria(dados.Vencimento, masculino("vencimento do repasse"))
	if err != nil {
		return nil, err
	}
	if dados.CategoriaFinanceiraID == uuid.Nil {
		return nil, domain.ErroValidacao("categoria financeira do repasse é obrigatória")
	}

	return &Repasse{
		ID:                    uuid.New(),
		RecebimentoID:         recebimento.ID,
		ValorCentavos:         recebimento.ValorCentavos,
		Vencimento:            vencimento,
		CategoriaFinanceiraID: dados.CategoriaFinanceiraID,
		UsuarioID:             usuarioID,
		CriadoEm:              agora,
	}, nil
}

func (r *Repasse) Observacao(titulo *Titulo) string {
	return fmt.Sprintf("repasse ao fomento do título %s/%d recebido em conta própria", titulo.Numero, titulo.Parcela)
}
