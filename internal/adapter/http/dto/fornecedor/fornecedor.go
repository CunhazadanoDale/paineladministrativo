package fornecedor

import (
	"time"

	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
	"github.com/google/uuid"
)

type DadosFornecedorRequest struct {
	IdentificacaoEstrangeira *string `json:"identificacao_estrangeira"`
	Nome                     string  `json:"nome"`
	NomeFantasia             *string `json:"nome_fantasia"`
	MoedaPadrao              string  `json:"moeda_padrao"`
	UF                       *string `json:"uf"`
	Cidade                   *string `json:"cidade"`
	ContatoNome              *string `json:"contato_nome"`
	Email                    *string `json:"email"`
	Telefone                 *string `json:"telefone"`
	Observacoes              string  `json:"observacoes"`
}

func (r DadosFornecedorRequest) ParaDados() portsin.DadosFornecedorInput {
	return portsin.DadosFornecedorInput{
		IdentificacaoEstrangeira: r.IdentificacaoEstrangeira,
		Nome:                     r.Nome,
		NomeFantasia:             r.NomeFantasia,
		MoedaPadrao:              r.MoedaPadrao,
		UF:                       r.UF,
		Cidade:                   r.Cidade,
		ContatoNome:              r.ContatoNome,
		Email:                    r.Email,
		Telefone:                 r.Telefone,
		Observacoes:              r.Observacoes,
	}
}

type CriarFornecedorRequest struct {
	DadosFornecedorRequest
	Pais      string  `json:"pais"`
	Documento *string `json:"documento"`
}

type AlterarFornecedorRequest struct {
	DadosFornecedorRequest
	Versao int `json:"versao"`
}

type SituacaoFornecedorRequest struct {
	Versao int `json:"versao"`
}

type FornecedorResponse struct {
	ID                       uuid.UUID `json:"id"`
	Pais                     string    `json:"pais"`
	Documento                *string   `json:"documento"`
	DocumentoFormatado       *string   `json:"documento_formatado"`
	TipoPessoa               *string   `json:"tipo_pessoa"`
	IdentificacaoEstrangeira *string   `json:"identificacao_estrangeira"`
	Nome                     string    `json:"nome"`
	NomeFantasia             *string   `json:"nome_fantasia"`
	MoedaPadrao              string    `json:"moeda_padrao"`
	UF                       *string   `json:"uf"`
	Cidade                   *string   `json:"cidade"`
	ContatoNome              *string   `json:"contato_nome"`
	Email                    *string   `json:"email"`
	Telefone                 *string   `json:"telefone"`
	Observacoes              string    `json:"observacoes"`
	Ativo                    bool      `json:"ativo"`
	Versao                   int       `json:"versao"`
	CriadoEm                 time.Time `json:"criado_em"`
	AtualizadoEm             time.Time `json:"atualizado_em"`
}

func NovoFornecedorResponse(item *domainfornecedor.Fornecedor) FornecedorResponse {
	resposta := FornecedorResponse{
		ID:                       item.ID,
		Pais:                     item.Pais.String(),
		IdentificacaoEstrangeira: item.IdentificacaoEstrangeira,
		Nome:                     item.Nome,
		NomeFantasia:             item.NomeFantasia,
		MoedaPadrao:              item.MoedaPadrao.String(),
		Cidade:                   item.Cidade,
		ContatoNome:              item.ContatoNome,
		Email:                    item.Email,
		Telefone:                 item.Telefone,
		Observacoes:              item.Observacoes,
		Ativo:                    item.Ativo,
		Versao:                   item.Versao,
		CriadoEm:                 item.CriadoEm,
		AtualizadoEm:             item.AtualizadoEm,
	}
	if !item.Documento.Vazio() {
		valor, formatado, pessoa := item.Documento.Valor(), item.Documento.Formatado(), item.Documento.Pessoa().String()
		resposta.Documento = &valor
		resposta.DocumentoFormatado = &formatado
		resposta.TipoPessoa = &pessoa
	}
	if item.UF != nil {
		uf := item.UF.String()
		resposta.UF = &uf
	}

	return resposta
}

func NovoFornecedorResponses(itens []*domainfornecedor.Fornecedor) []FornecedorResponse {
	respostas := make([]FornecedorResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoFornecedorResponse(item))
	}

	return respostas
}
