package cliente

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type GrupoRequest struct {
	Nome string `json:"nome"`
}

type GrupoResponse struct {
	ID           uuid.UUID `json:"id"`
	Nome         string    `json:"nome"`
	Ativo        bool      `json:"ativo"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

func NovoGrupoResponse(item *domaincliente.Grupo) GrupoResponse {
	return GrupoResponse{
		ID:           item.ID,
		Nome:         item.Nome,
		Ativo:        item.Ativo,
		CriadoEm:     item.CriadoEm,
		AtualizadoEm: item.AtualizadoEm,
	}
}

func NovoGrupoResponses(itens []*domaincliente.Grupo) []GrupoResponse {
	respostas := make([]GrupoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoGrupoResponse(item))
	}

	return respostas
}

type RepresentanteRequest struct {
	Nome                    string     `json:"nome"`
	Documento               *string    `json:"documento"`
	UsuarioID               *uuid.UUID `json:"usuario_id"`
	Email                   *string    `json:"email"`
	Telefone                *string    `json:"telefone"`
	ComissaoProntaEntregaBP int        `json:"comissao_pronta_entrega_bp"`
	ComissaoImportacaoBP    int        `json:"comissao_importacao_bp"`
}

func (r RepresentanteRequest) ParaDados() portsin.DadosRepresentanteInput {
	return portsin.DadosRepresentanteInput{
		Nome:                    r.Nome,
		Documento:               r.Documento,
		UsuarioVinculadoID:      r.UsuarioID,
		Email:                   r.Email,
		Telefone:                r.Telefone,
		ComissaoProntaEntregaBP: r.ComissaoProntaEntregaBP,
		ComissaoImportacaoBP:    r.ComissaoImportacaoBP,
	}
}

type DefinirFornecedorRequest struct {
	FornecedorID json.RawMessage `json:"fornecedor_id"`
}

func (r DefinirFornecedorRequest) Fornecedor() (*uuid.UUID, error) {
	valor := strings.TrimSpace(string(r.FornecedorID))
	if valor == "" {
		return nil, domain.ErroValidacao("fornecedor_id é obrigatório; envie null para tirar o fornecedor")
	}
	if valor == "null" {
		return nil, nil
	}

	var id uuid.UUID
	if err := json.Unmarshal(r.FornecedorID, &id); err != nil || id == uuid.Nil {
		return nil, domain.ErroValidacao("fornecedor_id inválido")
	}

	return &id, nil
}

type RepresentanteResponse struct {
	ID                      uuid.UUID  `json:"id"`
	Nome                    string     `json:"nome"`
	Documento               *string    `json:"documento"`
	DocumentoFormatado      *string    `json:"documento_formatado"`
	UsuarioID               *uuid.UUID `json:"usuario_id"`
	Email                   *string    `json:"email"`
	Telefone                *string    `json:"telefone"`
	ComissaoProntaEntregaBP int        `json:"comissao_pronta_entrega_bp"`
	ComissaoImportacaoBP    int        `json:"comissao_importacao_bp"`
	FornecedorID            *uuid.UUID `json:"fornecedor_id"`
	Ativo                   bool       `json:"ativo"`
	CriadoEm                time.Time  `json:"criado_em"`
	AtualizadoEm            time.Time  `json:"atualizado_em"`
}

func NovoRepresentanteResponse(item *domaincliente.Representante) RepresentanteResponse {
	resposta := RepresentanteResponse{
		ID:                      item.ID,
		Nome:                    item.Nome,
		UsuarioID:               item.UsuarioID,
		Email:                   item.Email,
		Telefone:                item.Telefone,
		ComissaoProntaEntregaBP: item.ComissaoProntaEntrega.PontosBase(),
		ComissaoImportacaoBP:    item.ComissaoImportacao.PontosBase(),
		FornecedorID:            item.FornecedorID,
		Ativo:                   item.Ativo,
		CriadoEm:                item.CriadoEm,
		AtualizadoEm:            item.AtualizadoEm,
	}
	if item.Documento != nil {
		valor, formatado := item.Documento.Valor(), item.Documento.Formatado()
		resposta.Documento = &valor
		resposta.DocumentoFormatado = &formatado
	}

	return resposta
}

func NovoRepresentanteResponses(itens []*domaincliente.Representante) []RepresentanteResponse {
	respostas := make([]RepresentanteResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoRepresentanteResponse(item))
	}

	return respostas
}

type ObraRequest struct {
	Nome        string  `json:"nome"`
	CNPJ        *string `json:"cnpj"`
	Logradouro  string  `json:"logradouro"`
	Numero      *string `json:"numero"`
	Complemento *string `json:"complemento"`
	Bairro      *string `json:"bairro"`
	Cidade      string  `json:"cidade"`
	UF          string  `json:"uf"`
	CEP         *string `json:"cep"`
}

func (r ObraRequest) ParaDados() portsin.DadosObraInput {
	return portsin.DadosObraInput{
		Nome:        r.Nome,
		CNPJ:        r.CNPJ,
		Logradouro:  r.Logradouro,
		Numero:      r.Numero,
		Complemento: r.Complemento,
		Bairro:      r.Bairro,
		Cidade:      r.Cidade,
		UF:          r.UF,
		CEP:         r.CEP,
	}
}

type ObraResponse struct {
	ID            uuid.UUID `json:"id"`
	ClienteID     uuid.UUID `json:"cliente_id"`
	Nome          string    `json:"nome"`
	CNPJ          *string   `json:"cnpj"`
	CNPJFormatado *string   `json:"cnpj_formatado"`
	Logradouro    string    `json:"logradouro"`
	Numero        *string   `json:"numero"`
	Complemento   *string   `json:"complemento"`
	Bairro        *string   `json:"bairro"`
	Cidade        string    `json:"cidade"`
	UF            string    `json:"uf"`
	CEP           *string   `json:"cep"`
	Ativo         bool      `json:"ativo"`
	CriadoEm      time.Time `json:"criado_em"`
	AtualizadoEm  time.Time `json:"atualizado_em"`
}

func NovaObraResponse(item *domaincliente.Obra) ObraResponse {
	resposta := ObraResponse{
		ID:           item.ID,
		ClienteID:    item.ClienteID,
		Nome:         item.Nome,
		Logradouro:   item.Endereco.Logradouro,
		Numero:       item.Endereco.Numero,
		Complemento:  item.Endereco.Complemento,
		Bairro:       item.Endereco.Bairro,
		Cidade:       item.Endereco.Cidade,
		UF:           item.Endereco.UF.String(),
		CEP:          item.Endereco.CEP,
		Ativo:        item.Ativo,
		CriadoEm:     item.CriadoEm,
		AtualizadoEm: item.AtualizadoEm,
	}
	if item.CNPJ != nil {
		valor, formatado := item.CNPJ.Digitos(), item.CNPJ.Formatado()
		resposta.CNPJ = &valor
		resposta.CNPJFormatado = &formatado
	}

	return resposta
}

func NovaObraResponses(itens []*domaincliente.Obra) []ObraResponse {
	respostas := make([]ObraResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaObraResponse(item))
	}

	return respostas
}
