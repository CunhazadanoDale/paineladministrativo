package empresa

import (
	"time"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	"github.com/google/uuid"
)

type CriarEmpresaRequest struct {
	CNPJ             string `json:"cnpj"`
	RazaoSocial      string `json:"razao_social"`
	NomeFantasia     string `json:"nome_fantasia"`
	UF               string `json:"uf"`
	RegimeTributario string `json:"regime_tributario"`
}

func (r CriarEmpresaRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarEmpresaInput {
	return portsin.CriarEmpresaInput{
		UsuarioID:        usuarioID,
		CNPJ:             r.CNPJ,
		RazaoSocial:      r.RazaoSocial,
		NomeFantasia:     r.NomeFantasia,
		UF:               r.UF,
		RegimeTributario: r.RegimeTributario,
	}
}

type AlterarEmpresaRequest struct {
	RazaoSocial      string `json:"razao_social"`
	NomeFantasia     string `json:"nome_fantasia"`
	UF               string `json:"uf"`
	RegimeTributario string `json:"regime_tributario"`
}

func (r AlterarEmpresaRequest) ParaInput(usuarioID, empresaID uuid.UUID) portsin.AlterarEmpresaInput {
	return portsin.AlterarEmpresaInput{
		UsuarioID:        usuarioID,
		EmpresaID:        empresaID,
		RazaoSocial:      r.RazaoSocial,
		NomeFantasia:     r.NomeFantasia,
		UF:               r.UF,
		RegimeTributario: r.RegimeTributario,
	}
}

type EmpresaResponse struct {
	ID               uuid.UUID `json:"id"`
	RazaoSocial      string    `json:"razao_social"`
	NomeFantasia     string    `json:"nome_fantasia"`
	CNPJ             string    `json:"cnpj"`
	CNPJFormatado    string    `json:"cnpj_formatado"`
	UF               string    `json:"uf"`
	RegimeTributario string    `json:"regime_tributario"`
	Ativa            bool      `json:"ativa"`
	CriadoEm         time.Time `json:"criado_em"`
	AtualizadoEm     time.Time `json:"atualizado_em"`
}

func NovaEmpresaResponse(item *domainempresa.Empresa) EmpresaResponse {
	return EmpresaResponse{
		ID:               item.ID,
		RazaoSocial:      item.RazaoSocial,
		NomeFantasia:     item.NomeFantasia,
		CNPJ:             item.CNPJ.Digitos(),
		CNPJFormatado:    item.CNPJ.Formatado(),
		UF:               item.UF.String(),
		RegimeTributario: item.RegimeTributario.String(),
		Ativa:            item.Ativa,
		CriadoEm:         item.CriadoEm,
		AtualizadoEm:     item.AtualizadoEm,
	}
}

func NovaEmpresaResponses(itens []*domainempresa.Empresa) []EmpresaResponse {
	respostas := make([]EmpresaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaEmpresaResponse(item))
	}

	return respostas
}
