package receber

import (
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	"github.com/google/uuid"
)

type DadosTituloRequest struct {
	Numero                string     `json:"numero"`
	Parcela               int        `json:"parcela"`
	Vencimento            string     `json:"vencimento"`
	ValorCentavos         int64      `json:"valor_centavos"`
	CategoriaFinanceiraID *uuid.UUID `json:"categoria_financeira_id"`
	Competencia           string     `json:"competencia"`
	Observacao            string     `json:"observacao"`
}

func (r DadosTituloRequest) ParaDados() (portsin.DadosEdicaoInput, error) {
	vencimento, err := ParaData(r.Vencimento, "vencimento")
	if err != nil {
		return portsin.DadosEdicaoInput{}, err
	}

	return portsin.DadosEdicaoInput{
		Numero:                r.Numero,
		Parcela:               r.Parcela,
		Vencimento:            vencimento,
		ValorCentavos:         r.ValorCentavos,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Competencia:           r.Competencia,
		Observacao:            r.Observacao,
	}, nil
}

type CriarTituloRequest struct {
	DadosTituloRequest
	EmpresaID       uuid.UUID  `json:"empresa_id"`
	ClienteID       uuid.UUID  `json:"cliente_id"`
	ContaBancariaID *uuid.UUID `json:"conta_bancaria_id"`
}

type EditarTituloRequest struct {
	DadosTituloRequest
	Versao int `json:"versao"`
}

type ProrrogarRequest struct {
	Versao     int    `json:"versao"`
	Vencimento string `json:"vencimento"`
	Motivo     string `json:"motivo"`
}

type EnviarAoBancoRequest struct {
	Versao          int       `json:"versao"`
	ContaBancariaID uuid.UUID `json:"conta_bancaria_id"`
}

type VersaoRequest struct {
	Versao int `json:"versao"`
}

type BoletoEnviadoRequest struct {
	Versao    int    `json:"versao"`
	Canal     string `json:"canal"`
	EnviadoEm string `json:"enviado_em"`
}

type CancelarRequest struct {
	Versao int    `json:"versao"`
	Motivo string `json:"motivo"`
}

type TituloDeEntradaRequest struct {
	Vencimento      string     `json:"vencimento"`
	ContaBancariaID *uuid.UUID `json:"conta_bancaria_id"`
}

type TituloResponse struct {
	ID                    uuid.UUID  `json:"id"`
	EmpresaID             uuid.UUID  `json:"empresa_id"`
	ClienteID             uuid.UUID  `json:"cliente_id"`
	Origem                string     `json:"origem"`
	OrigemID              *uuid.UUID `json:"origem_id"`
	Numero                string     `json:"numero"`
	Parcela               int        `json:"parcela"`
	Vencimento            string     `json:"vencimento"`
	ValorCentavos         int64      `json:"valor_centavos"`
	RecebidoCentavos      int64      `json:"recebido_centavos"`
	SaldoCentavos         int64      `json:"saldo_centavos"`
	Vencido               bool       `json:"vencido"`
	CategoriaFinanceiraID *uuid.UUID `json:"categoria_financeira_id"`
	Competencia           string     `json:"competencia"`
	ContaBancariaID       *uuid.UUID `json:"conta_bancaria_id"`
	Status                string     `json:"status"`
	BoletoEnviadoEm       *time.Time `json:"boleto_enviado_em"`
	BoletoCanal           *string    `json:"boleto_canal"`
	OrigemCanceladaEm     *time.Time `json:"origem_cancelada_em"`
	CanceladoMotivo       *string    `json:"cancelado_motivo"`
	Observacao            string     `json:"observacao"`
	UsuarioID             uuid.UUID  `json:"usuario_id"`
	Versao                int        `json:"versao"`
	CriadoEm              time.Time  `json:"criado_em"`
	AtualizadoEm          time.Time  `json:"atualizado_em"`
}

type HistoricoResponse struct {
	ID         uuid.UUID `json:"id"`
	UsuarioID  uuid.UUID `json:"usuario_id"`
	Evento     string    `json:"evento"`
	DeStatus   *string   `json:"de_status"`
	ParaStatus string    `json:"para_status"`
	Descricao  string    `json:"descricao"`
	CriadoEm   time.Time `json:"criado_em"`
}

func NovoTituloResponse(item *domainreceber.Titulo) TituloResponse {
	resposta := TituloResponse{
		ID:                    item.ID,
		EmpresaID:             item.EmpresaID,
		ClienteID:             item.ClienteID,
		Origem:                item.Origem.String(),
		OrigemID:              item.OrigemID,
		Numero:                item.Numero,
		Parcela:               item.Parcela,
		Vencimento:            formatarData(item.Vencimento),
		ValorCentavos:         item.ValorCentavos,
		RecebidoCentavos:      item.RecebidoCentavos,
		SaldoCentavos:         item.SaldoCentavos(),
		Vencido:               item.Vencido(domainreceber.Hoje(time.Now().UTC())),
		CategoriaFinanceiraID: item.CategoriaFinanceiraID,
		Competencia:           item.Competencia.String(),
		ContaBancariaID:       item.ContaBancariaID,
		Status:                item.Status.String(),
		BoletoEnviadoEm:       item.BoletoEnviadoEm,
		OrigemCanceladaEm:     item.OrigemCanceladaEm,
		CanceladoMotivo:       item.CanceladoMotivo,
		Observacao:            item.Observacao,
		UsuarioID:             item.UsuarioID,
		Versao:                item.Versao,
		CriadoEm:              item.CriadoEm,
		AtualizadoEm:          item.AtualizadoEm,
	}
	if item.BoletoCanal != nil {
		canal := item.BoletoCanal.String()
		resposta.BoletoCanal = &canal
	}

	return resposta
}

func NovoTituloResponses(itens []*domainreceber.Titulo) []TituloResponse {
	respostas := make([]TituloResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoTituloResponse(item))
	}

	return respostas
}

func NovoHistoricoResponses(itens []*domainreceber.Historico) []HistoricoResponse {
	respostas := make([]HistoricoResponse, 0, len(itens))
	for _, item := range itens {
		resposta := HistoricoResponse{
			ID:         item.ID,
			UsuarioID:  item.UsuarioID,
			Evento:     item.Evento.String(),
			ParaStatus: item.ParaStatus.String(),
			Descricao:  item.Descricao,
			CriadoEm:   item.CriadoEm,
		}
		if item.DeStatus != nil {
			de := item.DeStatus.String()
			resposta.DeStatus = &de
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}
