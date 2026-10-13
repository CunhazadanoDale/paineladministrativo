package cliente

import (
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type DadosClienteRequest struct {
	Nome                    string     `json:"nome"`
	NomeFantasia            *string    `json:"nome_fantasia"`
	Tipo                    string     `json:"tipo"`
	GrupoID                 *uuid.UUID `json:"grupo_id"`
	UF                      string     `json:"uf"`
	Cidade                  string     `json:"cidade"`
	EmpresaRelacionamentoID *uuid.UUID `json:"empresa_relacionamento_id"`
	RepresentanteID         *uuid.UUID `json:"representante_id"`
	EmailNF                 *string    `json:"email_nf"`
	Telefone                *string    `json:"telefone"`
	Observacoes             string     `json:"observacoes"`
	ExigeNumeroPedidoNaNF   bool       `json:"exige_numero_pedido_na_nf"`
	ProibeAntecipacao       bool       `json:"proibe_antecipacao"`
}

func (r DadosClienteRequest) ParaInput() portsin.DadosClienteInput {
	return portsin.DadosClienteInput{
		Nome:                    r.Nome,
		NomeFantasia:            r.NomeFantasia,
		Tipo:                    r.Tipo,
		GrupoID:                 r.GrupoID,
		UF:                      r.UF,
		Cidade:                  r.Cidade,
		EmpresaRelacionamentoID: r.EmpresaRelacionamentoID,
		RepresentanteID:         r.RepresentanteID,
		EmailNF:                 r.EmailNF,
		Telefone:                r.Telefone,
		Observacoes:             r.Observacoes,
		ExigeNumeroPedidoNaNF:   r.ExigeNumeroPedidoNaNF,
		ProibeAntecipacao:       r.ProibeAntecipacao,
	}
}

type CriarClienteRequest struct {
	Documento string `json:"documento"`
	DadosClienteRequest
}

func (r CriarClienteRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarClienteInput {
	return portsin.CriarClienteInput{
		UsuarioID: usuarioID,
		Documento: r.Documento,
		Dados:     r.DadosClienteRequest.ParaInput(),
	}
}

type AlterarClienteRequest struct {
	DadosClienteRequest
}

func (r AlterarClienteRequest) ParaInput(usuarioID, clienteID uuid.UUID) portsin.AlterarClienteInput {
	return portsin.AlterarClienteInput{
		UsuarioID: usuarioID,
		ClienteID: clienteID,
		Dados:     r.DadosClienteRequest.ParaInput(),
	}
}

type ClienteResponse struct {
	ID                      uuid.UUID  `json:"id"`
	Documento               string     `json:"documento"`
	DocumentoFormatado      string     `json:"documento_formatado"`
	TipoPessoa              string     `json:"tipo_pessoa"`
	Nome                    string     `json:"nome"`
	NomeFantasia            *string    `json:"nome_fantasia"`
	Tipo                    string     `json:"tipo"`
	GrupoID                 *uuid.UUID `json:"grupo_id"`
	UF                      string     `json:"uf"`
	Cidade                  string     `json:"cidade"`
	EmpresaRelacionamentoID *uuid.UUID `json:"empresa_relacionamento_id"`
	RepresentanteID         *uuid.UUID `json:"representante_id"`
	EmailNF                 *string    `json:"email_nf"`
	Telefone                *string    `json:"telefone"`
	Observacoes             string     `json:"observacoes"`
	ExigeNumeroPedidoNaNF   bool       `json:"exige_numero_pedido_na_nf"`
	ProibeAntecipacao       bool       `json:"proibe_antecipacao"`
	LeadID                  *uuid.UUID `json:"lead_id"`
	Ativo                   bool       `json:"ativo"`
	CriadoEm                time.Time  `json:"criado_em"`
	AtualizadoEm            time.Time  `json:"atualizado_em"`
}

func NovoClienteResponse(item *domaincliente.Cliente) ClienteResponse {
	return ClienteResponse{
		ID:                      item.ID,
		Documento:               item.Documento.Valor(),
		DocumentoFormatado:      item.Documento.Formatado(),
		TipoPessoa:              item.Documento.Pessoa().String(),
		Nome:                    item.Nome,
		NomeFantasia:            item.NomeFantasia,
		Tipo:                    item.Tipo.String(),
		GrupoID:                 item.GrupoID,
		UF:                      item.UF.String(),
		Cidade:                  item.Cidade,
		EmpresaRelacionamentoID: item.EmpresaRelacionamentoID,
		RepresentanteID:         item.RepresentanteID,
		EmailNF:                 item.EmailNF,
		Telefone:                item.Telefone,
		Observacoes:             item.Observacoes,
		ExigeNumeroPedidoNaNF:   item.ExigeNumeroPedidoNaNF,
		ProibeAntecipacao:       item.ProibeAntecipacao,
		LeadID:                  item.LeadID,
		Ativo:                   item.Ativo,
		CriadoEm:                item.CriadoEm,
		AtualizadoEm:            item.AtualizadoEm,
	}
}

func NovoClienteResponses(itens []*domaincliente.Cliente) []ClienteResponse {
	respostas := make([]ClienteResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoClienteResponse(item))
	}

	return respostas
}
