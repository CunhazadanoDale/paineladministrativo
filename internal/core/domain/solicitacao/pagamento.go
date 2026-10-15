package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type Pagamento struct {
	ID                   uuid.UUID  `db:"id"`
	SolicitacaoID        uuid.UUID  `db:"solicitacao_id"`
	ContaBancariaID      *uuid.UUID `db:"conta_bancaria_id"`
	UsuarioID            *uuid.UUID `db:"usuario_id"`
	ComprovanteArquivoID *uuid.UUID `db:"comprovante_arquivo_id"`
	Valor                Valor      `db:"valor"`
	PagoEm               time.Time  `db:"pago_em"`
	Estornado            bool       `db:"estornado"`
	Estorno              *Estorno   `db:"-"`
	CriadoEm             time.Time  `db:"criado_em"`
}

func ErroContaBancariaObrigatoria() error {
	return domain.ErroValidacao("conta bancária do pagamento é obrigatória")
}

func ErroContaBancariaNaoEncontrada() error {
	return domain.ErroValidacao("conta bancária não encontrada")
}

func ErroContaBancariaInativa() error {
	return domain.ErroValidacao("conta bancária inativa não recebe pagamento")
}

func ErroContaBancariaDeOutraEmpresa() error {
	return domain.ErroValidacao("conta bancária deve ser da empresa da solicitação")
}

func ErroEmpresaInativaNoPagamento() error {
	return domain.ErroValidacao("empresa inativa não recebe pagamento")
}

func ErroPagamentoNaoEncontrado() error {
	return domain.ErroNaoEncontrado("pagamento não encontrado")
}

func NovoPagamento(
	solicitacaoID uuid.UUID,
	contaBancariaID uuid.UUID,
	usuarioID uuid.UUID,
	valor Valor,
	comprovanteArquivoID *uuid.UUID,
	pagoEm, agora time.Time,
) (*Pagamento, error) {
	if contaBancariaID == uuid.Nil {
		return nil, ErroContaBancariaObrigatoria()
	}
	if pagoEm.IsZero() {
		pagoEm = agora
	}
	if pagoEm.After(agora) {
		return nil, domain.ErroValidacao("data do pagamento não pode estar no futuro")
	}

	return &Pagamento{
		ID:                   uuid.New(),
		SolicitacaoID:        solicitacaoID,
		ContaBancariaID:      &contaBancariaID,
		UsuarioID:            &usuarioID,
		ComprovanteArquivoID: comprovanteArquivoID,
		Valor:                valor,
		PagoEm:               pagoEm,
		CriadoEm:             agora,
	}, nil
}
