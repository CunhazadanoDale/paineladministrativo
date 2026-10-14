package cliente

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoNomeCliente  = 200
	tamanhoMaximoNomeFantasia = 120
	tamanhoMaximoCidade       = 120
	tamanhoMaximoObservacoes  = 2000
)

type Cliente struct {
	ID                      uuid.UUID
	Documento               Documento
	Nome                    string
	NomeFantasia            *string
	NomeBusca               string
	Tipo                    TipoCliente
	GrupoID                 *uuid.UUID
	UF                      domainempresa.UF
	Cidade                  string
	EmpresaRelacionamentoID *uuid.UUID
	RepresentanteID         *uuid.UUID
	EmailNF                 *string
	Telefone                *string
	Observacoes             string
	ExigeNumeroPedidoNaNF   bool
	ProibeAntecipacao       bool
	LeadID                  *uuid.UUID
	Ativo                   bool
	CriadoEm                time.Time
	AtualizadoEm            time.Time
}

type DadosCliente struct {
	Nome                    string
	NomeFantasia            *string
	Tipo                    TipoCliente
	GrupoID                 *uuid.UUID
	UF                      domainempresa.UF
	Cidade                  string
	EmpresaRelacionamentoID *uuid.UUID
	RepresentanteID         *uuid.UUID
	EmailNF                 *string
	Telefone                *string
	Observacoes             string
	ExigeNumeroPedidoNaNF   bool
	ProibeAntecipacao       bool
}

func NovoCliente(documento Documento, dados DadosCliente, leadID *uuid.UUID) (*Cliente, error) {
	if documento.Vazio() {
		return nil, domain.ErroValidacao("documento do cliente é obrigatório")
	}
	if leadID != nil && *leadID == uuid.Nil {
		leadID = nil
	}

	agora := time.Now().UTC()
	cliente := &Cliente{
		ID:        uuid.New(),
		Documento: documento,
		LeadID:    leadID,
		Ativo:     true,
		CriadoEm:  agora,
	}

	if err := cliente.Alterar(dados, agora); err != nil {
		return nil, err
	}

	return cliente, nil
}

func (c *Cliente) Alterar(dados DadosCliente, agora time.Time) error {
	validados, err := validarDadosCliente(c.Documento, dados)
	if err != nil {
		return err
	}

	c.Nome = validados.Nome
	c.NomeFantasia = validados.NomeFantasia
	c.NomeBusca = NomeDeBusca(validados.Nome)
	c.Tipo = validados.Tipo
	c.GrupoID = validados.GrupoID
	c.UF = validados.UF
	c.Cidade = validados.Cidade
	c.EmpresaRelacionamentoID = validados.EmpresaRelacionamentoID
	c.RepresentanteID = validados.RepresentanteID
	c.EmailNF = validados.EmailNF
	c.Telefone = validados.Telefone
	c.Observacoes = validados.Observacoes
	c.ExigeNumeroPedidoNaNF = validados.ExigeNumeroPedidoNaNF
	c.ProibeAntecipacao = validados.ProibeAntecipacao
	c.AtualizadoEm = agora

	return nil
}

func (c *Cliente) Ativar(agora time.Time) {
	c.Ativo = true
	c.AtualizadoEm = agora
}

func (c *Cliente) Desativar(agora time.Time) {
	c.Ativo = false
	c.AtualizadoEm = agora
}

func validarDadosCliente(documento Documento, dados DadosCliente) (DadosCliente, error) {
	var err error

	if dados.Nome, err = textoObrigatorio(dados.Nome, masculino("nome do cliente"), tamanhoMaximoNomeCliente); err != nil {
		return dados, err
	}
	if dados.NomeFantasia, err = textoOpcional(dados.NomeFantasia, "nome fantasia do cliente", tamanhoMaximoNomeFantasia); err != nil {
		return dados, err
	}
	if err = dados.Tipo.Validado(); err != nil {
		return dados, err
	}
	if (dados.Tipo == TipoClientePF) != (documento.Pessoa() == PessoaFisica) {
		return dados, domain.ErroValidacao("cliente do tipo pf exige CPF, e CPF exige o tipo pf")
	}
	if err = dados.UF.Validado(); err != nil {
		return dados, err
	}
	if dados.Cidade, err = textoObrigatorio(dados.Cidade, feminino("cidade do cliente"), tamanhoMaximoCidade); err != nil {
		return dados, err
	}
	if dados.EmailNF, err = emailOpcional(dados.EmailNF, "e-mail para NF"); err != nil {
		return dados, err
	}
	if dados.Telefone, err = textoOpcional(dados.Telefone, "telefone do cliente", tamanhoMaximoTelefone); err != nil {
		return dados, err
	}

	observacoes, err := textoOpcional(&dados.Observacoes, "observações do cliente", tamanhoMaximoObservacoes)
	if err != nil {
		return dados, err
	}
	dados.Observacoes = ""
	if observacoes != nil {
		dados.Observacoes = *observacoes
	}

	dados.GrupoID = idOpcional(dados.GrupoID)
	dados.EmpresaRelacionamentoID = idOpcional(dados.EmpresaRelacionamentoID)
	dados.RepresentanteID = idOpcional(dados.RepresentanteID)

	return dados, nil
}

func idOpcional(id *uuid.UUID) *uuid.UUID {
	if id == nil || *id == uuid.Nil {
		return nil
	}

	return id
}
