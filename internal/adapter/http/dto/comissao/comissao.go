package comissao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

const formatoData = "2006-01-02"

type DefinirPercentualRequest struct {
	PercentualBP *int `json:"percentual_bp"`
	Versao       *int `json:"versao"`
}

func (r DefinirPercentualRequest) PontosBase() (int, error) {
	if r.PercentualBP == nil {
		return 0, domain.ErroValidacao("percentual_bp é obrigatório")
	}

	return *r.PercentualBP, nil
}

type ApurarRequest struct {
	EmpresaID   *uuid.UUID `json:"empresa_id"`
	Competencia string     `json:"competencia"`
}

type PercentualResponse struct {
	PedidoID        uuid.UUID  `json:"pedido_id"`
	RepresentanteID uuid.UUID  `json:"representante_id"`
	PercentualBP    int        `json:"percentual_bp"`
	Origem          string     `json:"origem"`
	Definido        bool       `json:"definido"`
	Versao          *int       `json:"versao"`
	UsuarioID       *uuid.UUID `json:"usuario_id"`
	AtualizadoEm    *time.Time `json:"atualizado_em"`
}

type FalhaApuracaoResponse struct {
	EntregaID uuid.UUID `json:"entrega_id"`
	Motivo    string    `json:"motivo"`
}

type ResultadoApuracaoResponse struct {
	Apuradas       []uuid.UUID             `json:"apuradas"`
	JaExistentes   int                     `json:"ja_existentes"`
	Falhas         []FalhaApuracaoResponse `json:"falhas"`
	RestamEntregas bool                    `json:"restam_entregas"`
}

type EntregaPendenteResponse struct {
	EntregaID       uuid.UUID `json:"entrega_id"`
	PedidoID        uuid.UUID `json:"pedido_id"`
	PedidoNumero    int       `json:"pedido_numero"`
	EmpresaID       uuid.UUID `json:"empresa_id"`
	RepresentanteID uuid.UUID `json:"representante_id"`
	Modelo          string    `json:"modelo"`
	SaiuEm          string    `json:"saiu_em"`
}

type ComissaoResponse struct {
	ID              uuid.UUID  `json:"id"`
	EmpresaID       uuid.UUID  `json:"empresa_id"`
	RepresentanteID uuid.UUID  `json:"representante_id"`
	PedidoID        uuid.UUID  `json:"pedido_id"`
	EntregaID       uuid.UUID  `json:"entrega_id"`
	Competencia     string     `json:"competencia"`
	BaseCentavos    int64      `json:"base_centavos"`
	PercentualBP    int        `json:"percentual_bp"`
	ValorCentavos   int64      `json:"valor_centavos"`
	FechamentoID    *uuid.UUID `json:"fechamento_id"`
	UsuarioID       uuid.UUID  `json:"usuario_id"`
	CriadoEm        time.Time  `json:"criado_em"`
}

func NovoPercentualResponse(item *domaincomissao.PercentualPedido) PercentualResponse {
	resposta := PercentualResponse{
		PedidoID:        item.PedidoID,
		RepresentanteID: item.RepresentanteID,
		PercentualBP:    item.Percentual.PontosBase(),
		Origem:          item.Origem.String(),
		Definido:        item.Gravado(),
	}
	if item.Gravado() {
		versao, usuarioID, atualizadoEm := item.Versao, item.UsuarioID, item.AtualizadoEm
		resposta.Versao = &versao
		resposta.UsuarioID = &usuarioID
		resposta.AtualizadoEm = &atualizadoEm
	}

	return resposta
}

func NovoResultadoApuracaoResponse(resultado domaincomissao.ResultadoApuracao) ResultadoApuracaoResponse {
	falhas := make([]FalhaApuracaoResponse, 0, len(resultado.Falhas))
	for _, falha := range resultado.Falhas {
		falhas = append(falhas, FalhaApuracaoResponse{EntregaID: falha.EntregaID, Motivo: falha.Motivo})
	}
	apuradas := resultado.Apuradas
	if apuradas == nil {
		apuradas = []uuid.UUID{}
	}

	return ResultadoApuracaoResponse{
		Apuradas:       apuradas,
		JaExistentes:   resultado.JaExistentes,
		Falhas:         falhas,
		RestamEntregas: resultado.RestamEntregas,
	}
}

func NovoEntregaPendenteResponses(itens []*domaincomissao.EntregaParaApurar) []EntregaPendenteResponse {
	respostas := make([]EntregaPendenteResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, EntregaPendenteResponse{
			EntregaID:       item.EntregaID,
			PedidoID:        item.PedidoID,
			PedidoNumero:    item.PedidoNumero,
			EmpresaID:       item.EmpresaID,
			RepresentanteID: item.RepresentanteID,
			Modelo:          item.Modelo.String(),
			SaiuEm:          item.SaiuEm.Format(formatoData),
		})
	}

	return respostas
}

func NovoComissaoResponses(itens []*domaincomissao.Comissao) []ComissaoResponse {
	respostas := make([]ComissaoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, ComissaoResponse{
			ID:              item.ID,
			EmpresaID:       item.EmpresaID,
			RepresentanteID: item.RepresentanteID,
			PedidoID:        item.PedidoID,
			EntregaID:       item.EntregaID,
			Competencia:     item.Competencia.String(),
			BaseCentavos:    item.BaseCentavos,
			PercentualBP:    item.Percentual.PontosBase(),
			ValorCentavos:   item.ValorCentavos,
			FechamentoID:    item.FechamentoID,
			UsuarioID:       item.UsuarioID,
			CriadoEm:        item.CriadoEm,
		})
	}

	return respostas
}
