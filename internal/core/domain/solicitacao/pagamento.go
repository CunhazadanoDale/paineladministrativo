package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type Pagamento struct {
	ID                   uuid.UUID  `db:"id"`
	SolicitacaoID        uuid.UUID  `db:"solicitacao_id"`
	ComprovanteArquivoID *uuid.UUID `db:"comprovante_arquivo_id"`
	Valor                Valor      `db:"valor"`
	PagoEm               time.Time  `db:"pago_em"`
	CriadoEm             time.Time  `db:"criado_em"`
}

func NovoPagamento(solicitacaoID uuid.UUID, valor Valor, comprovanteArquivoID *uuid.UUID, pagoEm, agora time.Time) (*Pagamento, error) {
	if pagoEm.IsZero() {
		pagoEm = agora
	}
	if pagoEm.After(agora) {
		return nil, domain.ErroValidacao("data do pagamento não pode estar no futuro")
	}

	return &Pagamento{
		ID:                   uuid.New(),
		SolicitacaoID:        solicitacaoID,
		ComprovanteArquivoID: comprovanteArquivoID,
		Valor:                valor,
		PagoEm:               pagoEm,
		CriadoEm:             agora,
	}, nil
}
