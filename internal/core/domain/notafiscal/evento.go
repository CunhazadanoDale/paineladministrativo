package notafiscal

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type Evento struct {
	ID            uuid.UUID
	NotaFiscalID  uuid.UUID
	Tipo          TipoEvento
	Sequencia     int
	OcorridoEm    time.Time
	Protocolo     string
	Justificativa string
	ArquivoID     uuid.UUID
	UsuarioID     uuid.UUID
	CriadoEm      time.Time
}

func (n *NotaFiscal) AplicarEvento(documento *DocumentoEvento, arquivoID, usuarioID uuid.UUID, agora time.Time) (*Evento, error) {
	switch {
	case documento == nil:
		return nil, domain.ErroValidacao("documento do evento não informado")
	case arquivoID == uuid.Nil:
		return nil, domain.ErroValidacao("arquivo XML do evento não informado")
	case usuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário do evento não informado")
	case documento.Chave.Valor() != n.Chave.Valor():
		return nil, domain.ErroValidacao("o evento é de outra nota fiscal")
	case documento.AutorCNPJ != n.Chave.CNPJEmitente():
		return nil, domain.ErroValidacao("o evento precisa ser do emitente da nota")
	case n.Status == StatusCancelada && documento.Tipo == TipoEventoCancelamento:
		return nil, domain.ErroConflito("nota fiscal já cancelada")
	case n.Status == StatusCancelada:
		return nil, domain.ErroConflito("nota fiscal cancelada não recebe carta de correção")
	}

	momento := agora.UTC()
	evento := &Evento{
		ID:            uuid.New(),
		NotaFiscalID:  n.ID,
		Tipo:          documento.Tipo,
		Sequencia:     documento.Sequencia,
		OcorridoEm:    documento.OcorridoEm,
		Protocolo:     documento.Protocolo,
		Justificativa: documento.Justificativa,
		ArquivoID:     arquivoID,
		UsuarioID:     usuarioID,
		CriadoEm:      momento,
	}

	if documento.Tipo == TipoEventoCancelamento {
		n.Status = StatusCancelada
		n.AtualizadoEm = momento
	}

	return evento, nil
}
