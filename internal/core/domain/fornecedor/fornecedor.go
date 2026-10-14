package fornecedor

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

const (
	versaoInicial                         = 1
	tamanhoMaximoNome                     = 200
	tamanhoMaximoNomeFantasia             = 120
	tamanhoMaximoIdentificacaoEstrangeira = 40
	tamanhoMaximoCidade                   = 120
	tamanhoMaximoContatoNome              = 160
	tamanhoMaximoObservacoes              = 2000
)

type Fornecedor struct {
	ID                       uuid.UUID
	Pais                     Pais
	Documento                domaincliente.Documento
	IdentificacaoEstrangeira *string
	Nome                     string
	NomeFantasia             *string
	NomeBusca                string
	MoedaPadrao              Moeda
	UF                       *domainempresa.UF
	Cidade                   *string
	ContatoNome              *string
	Email                    *string
	Telefone                 *string
	Observacoes              string
	Ativo                    bool
	Versao                   int
	CriadoEm                 time.Time
	AtualizadoEm             time.Time
}

type DadosFornecedor struct {
	IdentificacaoEstrangeira *string
	Nome                     string
	NomeFantasia             *string
	MoedaPadrao              Moeda
	UF                       *domainempresa.UF
	Cidade                   *string
	ContatoNome              *string
	Email                    *string
	Telefone                 *string
	Observacoes              string
}

func NovoFornecedor(pais Pais, documento domaincliente.Documento, dados DadosFornecedor) (*Fornecedor, error) {
	if err := pais.Validado(); err != nil {
		return nil, err
	}
	if pais.Brasil() && documento.Vazio() {
		return nil, domain.ErroValidacao("documento é obrigatório para fornecedor do Brasil")
	}
	if !pais.Brasil() && !documento.Vazio() {
		return nil, domain.ErroValidacao("fornecedor de fora do Brasil não tem CPF ou CNPJ; use a identificação estrangeira")
	}

	agora := time.Now().UTC()
	fornecedor := &Fornecedor{
		ID:        uuid.New(),
		Pais:      pais,
		Documento: documento,
		Ativo:     true,
		Versao:    versaoInicial,
		CriadoEm:  agora,
	}

	if err := fornecedor.Alterar(dados, agora); err != nil {
		return nil, err
	}

	return fornecedor, nil
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("o fornecedor foi alterado por outra pessoa; recarregue e tente de novo")
}

func (f *Fornecedor) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão do fornecedor é obrigatória")
	}
	if informada != f.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (f *Fornecedor) Alterar(dados DadosFornecedor, agora time.Time) error {
	validados, err := validarDados(f.Pais, dados)
	if err != nil {
		return err
	}

	f.IdentificacaoEstrangeira = validados.IdentificacaoEstrangeira
	f.Nome = validados.Nome
	f.NomeFantasia = validados.NomeFantasia
	f.NomeBusca = domaincliente.NomeDeBusca(validados.Nome)
	f.MoedaPadrao = validados.MoedaPadrao
	f.UF = validados.UF
	f.Cidade = validados.Cidade
	f.ContatoNome = validados.ContatoNome
	f.Email = validados.Email
	f.Telefone = validados.Telefone
	f.Observacoes = validados.Observacoes
	f.AtualizadoEm = agora

	return nil
}

func (f *Fornecedor) Ativar(agora time.Time) bool {
	if f.Ativo {
		return false
	}

	f.Ativo = true
	f.AtualizadoEm = agora

	return true
}

func (f *Fornecedor) Desativar(agora time.Time) bool {
	if !f.Ativo {
		return false
	}

	f.Ativo = false
	f.AtualizadoEm = agora

	return true
}

func validarDados(pais Pais, dados DadosFornecedor) (DadosFornecedor, error) {
	var err error

	if dados.Nome, err = textoObrigatorio(dados.Nome, "nome do fornecedor", tamanhoMaximoNome); err != nil {
		return dados, err
	}
	if dados.NomeFantasia, err = textoOpcional(dados.NomeFantasia, "nome fantasia do fornecedor", tamanhoMaximoNomeFantasia); err != nil {
		return dados, err
	}
	if dados.IdentificacaoEstrangeira, err = identificacaoEstrangeira(pais, dados.IdentificacaoEstrangeira); err != nil {
		return dados, err
	}
	if err = dados.MoedaPadrao.Validado(); err != nil {
		return dados, err
	}
	if dados.UF, err = ufOpcional(dados.UF); err != nil {
		return dados, err
	}
	if dados.Cidade, err = textoOpcional(dados.Cidade, "cidade do fornecedor", tamanhoMaximoCidade); err != nil {
		return dados, err
	}
	if !pais.Brasil() && (dados.UF != nil || dados.Cidade != nil) {
		return dados, domain.ErroValidacao("uf e cidade só valem para fornecedor do Brasil")
	}
	if dados.ContatoNome, err = textoOpcional(dados.ContatoNome, "nome do contato do fornecedor", tamanhoMaximoContatoNome); err != nil {
		return dados, err
	}
	if dados.Email, err = emailOpcional(dados.Email, "e-mail do fornecedor"); err != nil {
		return dados, err
	}
	if dados.Telefone, err = textoOpcional(dados.Telefone, "telefone do fornecedor", tamanhoMaximoTelefone); err != nil {
		return dados, err
	}

	observacoes, err := textoOpcional(&dados.Observacoes, "observações do fornecedor", tamanhoMaximoObservacoes)
	if err != nil {
		return dados, err
	}
	dados.Observacoes = ""
	if observacoes != nil {
		dados.Observacoes = *observacoes
	}

	return dados, nil
}

func identificacaoEstrangeira(pais Pais, valor *string) (*string, error) {
	identificacao, err := textoOpcional(valor, "identificação estrangeira do fornecedor", tamanhoMaximoIdentificacaoEstrangeira)
	if err != nil || identificacao == nil {
		return nil, err
	}
	if pais.Brasil() {
		return nil, domain.ErroValidacao("identificação estrangeira só vale para fornecedor de fora do Brasil")
	}

	maiuscula := strings.ToUpper(*identificacao)

	return &maiuscula, nil
}

func ufOpcional(valor *domainempresa.UF) (*domainempresa.UF, error) {
	if valor == nil {
		return nil, nil
	}

	uf := domainempresa.UFDe(strings.ToUpper(strings.TrimSpace(valor.String())))
	if uf == "" {
		return nil, nil
	}
	if err := uf.Validado(); err != nil {
		return nil, err
	}

	return &uf, nil
}
