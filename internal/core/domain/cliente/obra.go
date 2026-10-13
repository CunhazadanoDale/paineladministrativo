package cliente

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoNomeObra    = 160
	tamanhoMaximoLogradouro  = 200
	tamanhoMaximoNumero      = 20
	tamanhoMaximoComplemento = 120
	tamanhoMaximoBairro      = 120
	tamanhoCEP               = 8
)

type Endereco struct {
	Logradouro  string
	Numero      *string
	Complemento *string
	Bairro      *string
	Cidade      string
	UF          domainempresa.UF
	CEP         *string
}

type Obra struct {
	ID           uuid.UUID
	ClienteID    uuid.UUID
	Nome         string
	CNPJ         *domainempresa.CNPJ
	Endereco     Endereco
	Ativo        bool
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

type DadosObra struct {
	Nome     string
	CNPJ     *domainempresa.CNPJ
	Endereco Endereco
}

func NovaObra(clienteID uuid.UUID, dados DadosObra) (*Obra, error) {
	if clienteID == uuid.Nil {
		return nil, domain.ErroValidacao("cliente da obra não informado")
	}

	agora := time.Now().UTC()
	obra := &Obra{ID: uuid.New(), ClienteID: clienteID, Ativo: true, CriadoEm: agora}

	if err := obra.Alterar(dados, agora); err != nil {
		return nil, err
	}

	return obra, nil
}

func (o *Obra) Alterar(dados DadosObra, agora time.Time) error {
	nome, err := textoObrigatorio(dados.Nome, masculino("nome da obra"), tamanhoMaximoNomeObra)
	if err != nil {
		return err
	}

	endereco, err := validarEndereco(dados.Endereco)
	if err != nil {
		return err
	}

	if dados.CNPJ != nil && dados.CNPJ.Vazio() {
		dados.CNPJ = nil
	}

	o.Nome = nome
	o.CNPJ = dados.CNPJ
	o.Endereco = endereco
	o.AtualizadoEm = agora

	return nil
}

func (o *Obra) Ativar(agora time.Time) {
	o.Ativo = true
	o.AtualizadoEm = agora
}

func (o *Obra) Desativar(agora time.Time) {
	o.Ativo = false
	o.AtualizadoEm = agora
}

func validarEndereco(endereco Endereco) (Endereco, error) {
	var err error

	if endereco.Logradouro, err = textoObrigatorio(endereco.Logradouro, masculino("logradouro da obra"), tamanhoMaximoLogradouro); err != nil {
		return endereco, err
	}
	if endereco.Numero, err = textoOpcional(endereco.Numero, "número da obra", tamanhoMaximoNumero); err != nil {
		return endereco, err
	}
	if endereco.Complemento, err = textoOpcional(endereco.Complemento, "complemento da obra", tamanhoMaximoComplemento); err != nil {
		return endereco, err
	}
	if endereco.Bairro, err = textoOpcional(endereco.Bairro, "bairro da obra", tamanhoMaximoBairro); err != nil {
		return endereco, err
	}
	if endereco.Cidade, err = textoObrigatorio(endereco.Cidade, feminino("cidade da obra"), tamanhoMaximoCidade); err != nil {
		return endereco, err
	}
	if err = endereco.UF.Validado(); err != nil {
		return endereco, err
	}
	if endereco.CEP, err = cepOpcional(endereco.CEP); err != nil {
		return endereco, err
	}

	return endereco, nil
}

func cepOpcional(valor *string) (*string, error) {
	if valor == nil {
		return nil, nil
	}

	cep := semMascara(*valor)
	if cep == "" {
		return nil, nil
	}
	if len(cep) != tamanhoCEP || !somenteDigitos(cep) {
		return nil, domain.ErroValidacao("cep da obra deve ter 8 dígitos")
	}

	return &cep, nil
}
