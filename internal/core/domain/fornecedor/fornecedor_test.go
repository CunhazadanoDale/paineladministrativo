package fornecedor_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

func texto(valor string) *string {
	return &valor
}

func uf(valor string) *domainempresa.UF {
	sigla := domainempresa.UFDe(valor)
	return &sigla
}

func documentoDeTeste(t *testing.T, valor string) domaincliente.Documento {
	t.Helper()

	documento, err := domaincliente.NovoDocumento(valor)
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}

	return documento
}

func dadosNacionais() domainfornecedor.DadosFornecedor {
	return domainfornecedor.DadosFornecedor{
		Nome:         "  Cerâmica  Paraíba LTDA ",
		NomeFantasia: texto(" Cerâmica PB "),
		MoedaPadrao:  domainfornecedor.MoedaReal,
		UF:           uf(" pb "),
		Cidade:       texto(" João Pessoa "),
		ContatoNome:  texto("   "),
		Email:        texto(" vendas@ceramicapb.com.br "),
		Telefone:     texto(" (83) 3333-0000 "),
		Observacoes:  "  entrega só às terças ",
	}
}

func dadosEstrangeiros() domainfornecedor.DadosFornecedor {
	return domainfornecedor.DadosFornecedor{
		IdentificacaoEstrangeira: texto(" 91330200ma2abc "),
		Nome:                     "Foshan Ceramics Co., Ltd.",
		MoedaPadrao:              domainfornecedor.MoedaDolar,
		ContatoNome:              texto(" Li Wei "),
	}
}

func novoNacional(t *testing.T) *domainfornecedor.Fornecedor {
	t.Helper()

	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, documentoDeTeste(t, "11.222.333/0001-81"), dadosNacionais())
	if err != nil {
		t.Fatalf("NovoFornecedor falhou: %v", err)
	}

	return fornecedor
}

func novoEstrangeiro(t *testing.T) *domainfornecedor.Fornecedor {
	t.Helper()

	fornecedor, err := domainfornecedor.NovoFornecedor("CN", domaincliente.Documento{}, dadosEstrangeiros())
	if err != nil {
		t.Fatalf("NovoFornecedor falhou: %v", err)
	}

	return fornecedor
}

func TestNovoFornecedorNacionalNormalizaENasceAtivoNaVersaoUm(t *testing.T) {
	fornecedor := novoNacional(t)

	if fornecedor.ID == uuid.Nil || !fornecedor.Ativo || fornecedor.Versao != 1 {
		t.Errorf("fornecedor criado = %+v", fornecedor)
	}
	if fornecedor.Documento.Valor() != "11222333000181" || fornecedor.Documento.Pessoa() != domaincliente.PessoaJuridica {
		t.Errorf("documento = %q (%q)", fornecedor.Documento.Valor(), fornecedor.Documento.Pessoa())
	}
	if fornecedor.Nome != "Cerâmica  Paraíba LTDA" || fornecedor.NomeBusca != "ceramica paraiba ltda" {
		t.Errorf("nome %q e nome de busca %q", fornecedor.Nome, fornecedor.NomeBusca)
	}
	if fornecedor.NomeFantasia == nil || *fornecedor.NomeFantasia != "Cerâmica PB" {
		t.Errorf("nome fantasia = %v", fornecedor.NomeFantasia)
	}
	if fornecedor.UF == nil || *fornecedor.UF != "PB" || fornecedor.Cidade == nil || *fornecedor.Cidade != "João Pessoa" {
		t.Errorf("uf %v e cidade %v", fornecedor.UF, fornecedor.Cidade)
	}
	if fornecedor.ContatoNome != nil || fornecedor.Email == nil || *fornecedor.Email != "vendas@ceramicapb.com.br" {
		t.Errorf("contato %v e e-mail %v", fornecedor.ContatoNome, fornecedor.Email)
	}
	if fornecedor.Telefone == nil || *fornecedor.Telefone != "(83) 3333-0000" {
		t.Errorf("telefone = %v", fornecedor.Telefone)
	}
	if fornecedor.Observacoes != "entrega só às terças" || fornecedor.IdentificacaoEstrangeira != nil {
		t.Errorf("observações %q e identificação %v", fornecedor.Observacoes, fornecedor.IdentificacaoEstrangeira)
	}
	if fornecedor.CriadoEm.IsZero() || !fornecedor.AtualizadoEm.Equal(fornecedor.CriadoEm) {
		t.Errorf("criado em %v e atualizado em %v", fornecedor.CriadoEm, fornecedor.AtualizadoEm)
	}
}

func TestNovoFornecedorNacionalAceitaCPF(t *testing.T) {
	fornecedor, err := domainfornecedor.NovoFornecedor(
		domainfornecedor.PaisBrasil,
		documentoDeTeste(t, "529.982.247-25"),
		domainfornecedor.DadosFornecedor{Nome: "José Fretes", MoedaPadrao: domainfornecedor.MoedaReal},
	)
	if err != nil {
		t.Fatalf("NovoFornecedor falhou: %v", err)
	}
	if fornecedor.Documento.Pessoa() != domaincliente.PessoaFisica || fornecedor.UF != nil || fornecedor.Cidade != nil {
		t.Errorf("fornecedor pf = %+v", fornecedor)
	}
}

func TestNovoFornecedorEstrangeiroSemDocumento(t *testing.T) {
	fornecedor := novoEstrangeiro(t)

	if !fornecedor.Documento.Vazio() || fornecedor.Documento.Pessoa() != "" {
		t.Errorf("documento do estrangeiro = %+v", fornecedor.Documento)
	}
	if fornecedor.IdentificacaoEstrangeira == nil || *fornecedor.IdentificacaoEstrangeira != "91330200MA2ABC" {
		t.Errorf("identificação estrangeira = %v", fornecedor.IdentificacaoEstrangeira)
	}
	if fornecedor.MoedaPadrao != domainfornecedor.MoedaDolar || fornecedor.ContatoNome == nil || *fornecedor.ContatoNome != "Li Wei" {
		t.Errorf("moeda %q e contato %v", fornecedor.MoedaPadrao, fornecedor.ContatoNome)
	}
}

func TestNovoFornecedorEstrangeiroSemIdentificacao(t *testing.T) {
	dados := dadosEstrangeiros()
	dados.IdentificacaoEstrangeira = texto("   ")

	fornecedor, err := domainfornecedor.NovoFornecedor("US", domaincliente.Documento{}, dados)
	if err != nil {
		t.Fatalf("NovoFornecedor falhou: %v", err)
	}
	if fornecedor.IdentificacaoEstrangeira != nil {
		t.Errorf("identificação deveria ser nula, veio %q", *fornecedor.IdentificacaoEstrangeira)
	}
}

func TestNovoFornecedorRecusaRegrasDePaisEDocumento(t *testing.T) {
	cnpj := documentoDeTeste(t, "11.222.333/0001-81")

	nacionalComIdentificacao := dadosNacionais()
	nacionalComIdentificacao.IdentificacaoEstrangeira = texto("ABC123")

	estrangeiroComUF := dadosEstrangeiros()
	estrangeiroComUF.UF = uf("SP")

	estrangeiroComCidade := dadosEstrangeiros()
	estrangeiroComCidade.Cidade = texto("Foshan")

	casos := []struct {
		nome      string
		pais      domainfornecedor.Pais
		documento domaincliente.Documento
		dados     domainfornecedor.DadosFornecedor
		mensagem  string
	}{
		{"brasil sem documento", domainfornecedor.PaisBrasil, domaincliente.Documento{}, dadosNacionais(), "documento é obrigatório"},
		{"estrangeiro com documento", "CN", cnpj, dadosEstrangeiros(), "não tem CPF ou CNPJ"},
		{"país inválido", "CHN", domaincliente.Documento{}, dadosEstrangeiros(), "código ISO"},
		{"brasil com identificação estrangeira", domainfornecedor.PaisBrasil, cnpj, nacionalComIdentificacao, "só vale para fornecedor de fora"},
		{"estrangeiro com uf", "CN", domaincliente.Documento{}, estrangeiroComUF, "uf e cidade só valem"},
		{"estrangeiro com cidade", "CN", domaincliente.Documento{}, estrangeiroComCidade, "uf e cidade só valem"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := domainfornecedor.NovoFornecedor(caso.pais, caso.documento, caso.dados)
			if !errors.Is(err, domain.ErrValidacao) || !strings.Contains(err.Error(), caso.mensagem) {
				t.Errorf("erro = %v, esperado validação com %q", err, caso.mensagem)
			}
		})
	}
}

func TestNovoFornecedorRecusaDadosInvalidos(t *testing.T) {
	cnpj := documentoDeTeste(t, "11.222.333/0001-81")

	casos := map[string]func(*domainfornecedor.DadosFornecedor){
		"nome vazio":          func(d *domainfornecedor.DadosFornecedor) { d.Nome = "   " },
		"nome longo":          func(d *domainfornecedor.DadosFornecedor) { d.Nome = strings.Repeat("á", 201) },
		"nome fantasia longo": func(d *domainfornecedor.DadosFornecedor) { d.NomeFantasia = texto(strings.Repeat("a", 121)) },
		"moeda fora da lista": func(d *domainfornecedor.DadosFornecedor) { d.MoedaPadrao = "GBP" },
		"moeda vazia":         func(d *domainfornecedor.DadosFornecedor) { d.MoedaPadrao = "" },
		"uf inválida":         func(d *domainfornecedor.DadosFornecedor) { d.UF = uf("XX") },
		"cidade longa":        func(d *domainfornecedor.DadosFornecedor) { d.Cidade = texto(strings.Repeat("a", 121)) },
		"contato longo":       func(d *domainfornecedor.DadosFornecedor) { d.ContatoNome = texto(strings.Repeat("a", 161)) },
		"e-mail sem arroba":   func(d *domainfornecedor.DadosFornecedor) { d.Email = texto("vendas.ceramica.com") },
		"e-mail com espaço":   func(d *domainfornecedor.DadosFornecedor) { d.Email = texto("ven das@ceramica.com") },
		"telefone longo":      func(d *domainfornecedor.DadosFornecedor) { d.Telefone = texto(strings.Repeat("9", 41)) },
		"observações longas":  func(d *domainfornecedor.DadosFornecedor) { d.Observacoes = strings.Repeat("a", 2001) },
	}

	for nome, alterar := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosNacionais()
			alterar(&dados)

			if _, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, cnpj, dados); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v, esperado validação", err)
			}
		})
	}
}

func TestNovoFornecedorRecusaIdentificacaoEstrangeiraLonga(t *testing.T) {
	dados := dadosEstrangeiros()
	dados.IdentificacaoEstrangeira = texto(strings.Repeat("A", 41))

	if _, err := domainfornecedor.NovoFornecedor("CN", domaincliente.Documento{}, dados); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro = %v, esperado validação", err)
	}
}

func TestAlterarMantemPaisDocumentoEVersao(t *testing.T) {
	fornecedor := novoNacional(t)
	depois := fornecedor.CriadoEm.Add(time.Hour)

	err := fornecedor.Alterar(domainfornecedor.DadosFornecedor{
		Nome:        "Cerâmica Nordeste",
		MoedaPadrao: domainfornecedor.MoedaEuro,
	}, depois)
	if err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}

	if fornecedor.Pais != domainfornecedor.PaisBrasil || fornecedor.Documento.Valor() != "11222333000181" || fornecedor.Versao != 1 {
		t.Errorf("país, documento ou versão mudaram: %+v", fornecedor)
	}
	if fornecedor.NomeBusca != "ceramica nordeste" || fornecedor.MoedaPadrao != domainfornecedor.MoedaEuro {
		t.Errorf("nome de busca %q e moeda %q", fornecedor.NomeBusca, fornecedor.MoedaPadrao)
	}
	if fornecedor.UF != nil || fornecedor.Cidade != nil || fornecedor.Email != nil || fornecedor.Observacoes != "" {
		t.Errorf("campos opcionais não foram limpos: %+v", fornecedor)
	}
	if !fornecedor.AtualizadoEm.Equal(depois) {
		t.Errorf("atualizado em %v, esperado %v", fornecedor.AtualizadoEm, depois)
	}
}

func TestAlterarInvalidoNaoMudaOFornecedor(t *testing.T) {
	fornecedor := novoEstrangeiro(t)
	antes := *fornecedor

	dados := dadosEstrangeiros()
	dados.Nome = "Outro nome"
	dados.UF = uf("SP")

	if err := fornecedor.Alterar(dados, time.Now()); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("erro = %v, esperado validação", err)
	}
	if fornecedor.Nome != antes.Nome || !fornecedor.AtualizadoEm.Equal(antes.AtualizadoEm) {
		t.Errorf("fornecedor mudou depois de alteração inválida: %+v", fornecedor)
	}
}

func TestAtivarEDesativarInformamSeMudaram(t *testing.T) {
	fornecedor := novoEstrangeiro(t)
	criado := fornecedor.AtualizadoEm
	depois := criado.Add(time.Minute)

	if fornecedor.Ativar(depois) || !fornecedor.AtualizadoEm.Equal(criado) {
		t.Error("ativar fornecedor já ativo não deveria mudar nada")
	}
	if !fornecedor.Desativar(depois) || fornecedor.Ativo || !fornecedor.AtualizadoEm.Equal(depois) {
		t.Errorf("desativar deveria mudar: %+v", fornecedor)
	}
	if fornecedor.Desativar(depois.Add(time.Minute)) || !fornecedor.AtualizadoEm.Equal(depois) {
		t.Error("desativar fornecedor já inativo não deveria mudar nada")
	}
	if !fornecedor.Ativar(depois.Add(time.Hour)) || !fornecedor.Ativo {
		t.Errorf("ativar deveria mudar: %+v", fornecedor)
	}
	if fornecedor.Versao != 1 {
		t.Errorf("versão = %d; quem avança a versão é a gravação", fornecedor.Versao)
	}
}

func TestConferirVersao(t *testing.T) {
	fornecedor := &domainfornecedor.Fornecedor{Versao: 3}

	if err := fornecedor.ConferirVersao(3); err != nil {
		t.Errorf("versão igual deveria passar: %v", err)
	}
	for _, informada := range []int{0, -1} {
		if err := fornecedor.ConferirVersao(informada); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("ConferirVersao(%d) erro = %v, esperado validação", informada, err)
		}
	}
	for _, informada := range []int{2, 4} {
		err := fornecedor.ConferirVersao(informada)
		if !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "alterado por outra pessoa") {
			t.Errorf("ConferirVersao(%d) erro = %v, esperado conflito", informada, err)
		}
	}
}
