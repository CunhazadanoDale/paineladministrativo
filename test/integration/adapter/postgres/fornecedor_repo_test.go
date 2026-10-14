package postgres_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/fornecedor"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
)

type cenarioBancoFornecedores struct {
	ctx          context.Context
	fornecedores *postgres.FornecedorRepository
}

func novoCenarioFornecedores(t *testing.T) *cenarioBancoFornecedores {
	t.Helper()

	return &cenarioBancoFornecedores{
		ctx:          context.Background(),
		fornecedores: postgres.NewFornecedorRepository(helpers.BancoDoTeste(t)),
	}
}

func ufDe(valor string) *domainempresa.UF {
	uf := domainempresa.UFDe(valor)
	return &uf
}

func (c *cenarioBancoFornecedores) nacional(t *testing.T, documento, nome string) *domainfornecedor.Fornecedor {
	t.Helper()

	valor, err := domaincliente.NovoDocumento(documento)
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}

	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, valor, domainfornecedor.DadosFornecedor{
		Nome:        nome,
		MoedaPadrao: domainfornecedor.MoedaReal,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}

	return fornecedor
}

func (c *cenarioBancoFornecedores) estrangeiro(t *testing.T, pais domainfornecedor.Pais, identificacao, nome string, moeda domainfornecedor.Moeda) *domainfornecedor.Fornecedor {
	t.Helper()

	dados := domainfornecedor.DadosFornecedor{Nome: nome, MoedaPadrao: moeda}
	if identificacao != "" {
		dados.IdentificacaoEstrangeira = textoDe(identificacao)
	}

	fornecedor, err := domainfornecedor.NovoFornecedor(pais, domaincliente.Documento{}, dados)
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}

	return fornecedor
}

func (c *cenarioBancoFornecedores) criar(t *testing.T, fornecedor *domainfornecedor.Fornecedor) *domainfornecedor.Fornecedor {
	t.Helper()

	if _, err := c.fornecedores.Criar(c.ctx, fornecedor); err != nil {
		t.Fatalf("criação do fornecedor %q falhou: %v", fornecedor.Nome, err)
	}

	return fornecedor
}

func (c *cenarioBancoFornecedores) obter(t *testing.T, id uuid.UUID) *domainfornecedor.Fornecedor {
	t.Helper()

	fornecedor, err := c.fornecedores.Obter(c.ctx, id)
	if err != nil || fornecedor == nil {
		t.Fatalf("Obter(%s) = %v, %v", id, fornecedor, err)
	}

	return fornecedor
}

func (c *cenarioBancoFornecedores) nomesListados(t *testing.T, filtro portsout.FornecedorFiltro) []string {
	t.Helper()

	itens, err := c.fornecedores.Listar(c.ctx, filtro)
	if err != nil {
		t.Fatalf("Listar falhou: %v", err)
	}

	nomes := make([]string, 0, len(itens))
	for _, item := range itens {
		nomes = append(nomes, item.Nome)
	}

	return nomes
}

func TestFornecedorNacionalPersisteTodosOsCampos(t *testing.T) {
	c := novoCenarioFornecedores(t)

	documento, err := domaincliente.NovoDocumento("11.222.333/0001-81")
	if err != nil {
		t.Fatalf("documento inválido: %v", err)
	}
	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, documento, domainfornecedor.DadosFornecedor{
		Nome:         "Cerâmica Paraíba LTDA",
		NomeFantasia: textoDe("Cerâmica PB"),
		MoedaPadrao:  domainfornecedor.MoedaReal,
		UF:           ufDe("PB"),
		Cidade:       textoDe("João Pessoa"),
		ContatoNome:  textoDe("Maria"),
		Email:        textoDe("vendas@ceramicapb.com.br"),
		Telefone:     textoDe("(83) 3333-0000"),
		Observacoes:  "entrega às terças",
	})
	if err != nil {
		t.Fatalf("fornecedor inválido: %v", err)
	}
	c.criar(t, fornecedor)

	salvo := c.obter(t, fornecedor.ID)
	if salvo.Pais != domainfornecedor.PaisBrasil || salvo.Documento.Valor() != "11222333000181" || salvo.Documento.Pessoa() != domaincliente.PessoaJuridica {
		t.Errorf("país e documento = %q, %+v", salvo.Pais, salvo.Documento)
	}
	if salvo.Nome != "Cerâmica Paraíba LTDA" || *salvo.NomeFantasia != "Cerâmica PB" || salvo.NomeBusca != "ceramica paraiba ltda" {
		t.Errorf("nomes = %q, %v, %q", salvo.Nome, salvo.NomeFantasia, salvo.NomeBusca)
	}
	if salvo.UF == nil || *salvo.UF != "PB" || *salvo.Cidade != "João Pessoa" || salvo.IdentificacaoEstrangeira != nil {
		t.Errorf("endereço = %v, %v e identificação %v", salvo.UF, salvo.Cidade, salvo.IdentificacaoEstrangeira)
	}
	if *salvo.ContatoNome != "Maria" || *salvo.Email != "vendas@ceramicapb.com.br" || *salvo.Telefone != "(83) 3333-0000" {
		t.Errorf("contato = %v, %v, %v", salvo.ContatoNome, salvo.Email, salvo.Telefone)
	}
	if salvo.MoedaPadrao != domainfornecedor.MoedaReal || salvo.Observacoes != "entrega às terças" || !salvo.Ativo || salvo.Versao != 1 {
		t.Errorf("fornecedor salvo = %+v", salvo)
	}
	if diferenca := salvo.CriadoEm.Sub(fornecedor.CriadoEm).Abs(); diferenca > time.Millisecond {
		t.Errorf("criado em %v, esperado %v", salvo.CriadoEm, fornecedor.CriadoEm)
	}

	porDocumento, err := c.fornecedores.ObterPorDocumento(c.ctx, "11222333000181")
	if err != nil || porDocumento == nil || porDocumento.ID != fornecedor.ID {
		t.Errorf("ObterPorDocumento = %v, %v", porDocumento, err)
	}
	if nenhum, err := c.fornecedores.ObterPorDocumento(c.ctx, "52998224725"); err != nil || nenhum != nil {
		t.Errorf("ObterPorDocumento inexistente = %v, %v", nenhum, err)
	}
}

func TestFornecedorEstrangeiroPersisteSemDocumento(t *testing.T) {
	c := novoCenarioFornecedores(t)

	fornecedor := c.criar(t, c.estrangeiro(t, "CN", "91330200ma2abc", "Foshan Ceramics", domainfornecedor.MoedaDolar))
	semIdentificacao := c.criar(t, c.estrangeiro(t, "US", "", "Ohio Tools", domainfornecedor.MoedaDolar))

	salvo := c.obter(t, fornecedor.ID)
	if !salvo.Documento.Vazio() || salvo.Documento.Pessoa() != "" || salvo.UF != nil || salvo.Cidade != nil {
		t.Errorf("estrangeiro salvo com documento ou endereço: %+v", salvo)
	}
	if salvo.Pais != "CN" || salvo.IdentificacaoEstrangeira == nil || *salvo.IdentificacaoEstrangeira != "91330200MA2ABC" {
		t.Errorf("país %q e identificação %v", salvo.Pais, salvo.IdentificacaoEstrangeira)
	}
	if outro := c.obter(t, semIdentificacao.ID); outro.IdentificacaoEstrangeira != nil {
		t.Errorf("identificação deveria ser nula: %v", *outro.IdentificacaoEstrangeira)
	}

	encontrado, err := c.fornecedores.ObterPorIdentificacaoEstrangeira(c.ctx, "CN", "91330200MA2ABC")
	if err != nil || encontrado == nil || encontrado.ID != fornecedor.ID {
		t.Errorf("ObterPorIdentificacaoEstrangeira = %v, %v", encontrado, err)
	}
	if outroPais, err := c.fornecedores.ObterPorIdentificacaoEstrangeira(c.ctx, "US", "91330200MA2ABC"); err != nil || outroPais != nil {
		t.Errorf("identificação de outro país = %v, %v", outroPais, err)
	}
}

func TestFornecedorRespeitaUnicidade(t *testing.T) {
	c := novoCenarioFornecedores(t)

	c.criar(t, c.nacional(t, "11.222.333/0001-81", "Cerâmica A"))
	if _, err := c.fornecedores.Criar(c.ctx, c.nacional(t, "11222333000181", "Cerâmica B")); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("documento repetido: erro = %v, esperado conflito", err)
	}

	c.criar(t, c.estrangeiro(t, "CN", "ABC123", "Foshan A", domainfornecedor.MoedaDolar))
	if _, err := c.fornecedores.Criar(c.ctx, c.estrangeiro(t, "CN", "abc123", "Foshan B", domainfornecedor.MoedaDolar)); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("identificação repetida no país: erro = %v, esperado conflito", err)
	}
	c.criar(t, c.estrangeiro(t, "US", "ABC123", "Ohio", domainfornecedor.MoedaDolar))

	c.criar(t, c.estrangeiro(t, "CN", "", "Sem identificação 1", domainfornecedor.MoedaYuanChines))
	c.criar(t, c.estrangeiro(t, "CN", "", "Sem identificação 2", domainfornecedor.MoedaYuanChines))
}

func TestFornecedorBancoRecusaRegrasQueODominioGarante(t *testing.T) {
	c := novoCenarioFornecedores(t)

	estrangeiroComDocumento := c.estrangeiro(t, "CN", "", "Foshan", domainfornecedor.MoedaDolar)
	estrangeiroComDocumento.Documento = domaincliente.DocumentoDe("11222333000181")

	nacionalSemDocumento := c.nacional(t, "11.222.333/0001-81", "Cerâmica")
	nacionalSemDocumento.Documento = domaincliente.Documento{}

	estrangeiroComUF := c.estrangeiro(t, "CN", "", "Foshan UF", domainfornecedor.MoedaDolar)
	estrangeiroComUF.UF = ufDe("SP")

	for nome, fornecedor := range map[string]*domainfornecedor.Fornecedor{
		"estrangeiro com documento": estrangeiroComDocumento,
		"nacional sem documento":    nacionalSemDocumento,
		"estrangeiro com uf":        estrangeiroComUF,
	} {
		if _, err := c.fornecedores.Criar(c.ctx, fornecedor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado validação", nome, err)
		}
	}
}

func TestFornecedorListaComBuscaFiltrosEPaginacao(t *testing.T) {
	c := novoCenarioFornecedores(t)

	c.criar(t, c.nacional(t, "11.222.333/0001-81", "Cerâmica Paraíba"))
	c.criar(t, c.nacional(t, "529.982.247-25", "José Fretes"))
	c.criar(t, c.estrangeiro(t, "CN", "91330200MA2", "Foshan Ceramics", domainfornecedor.MoedaDolar))
	inativo := c.criar(t, c.estrangeiro(t, "CN", "", "Guangzhou Tiles", domainfornecedor.MoedaYuanChines))
	c.criar(t, c.estrangeiro(t, "IT", "IT0123", "Ceramiche Italia", domainfornecedor.MoedaEuro))

	inativo.Desativar(time.Now().UTC())
	if ok, err := c.fornecedores.Atualizar(c.ctx, inativo); err != nil || !ok {
		t.Fatalf("desativar falhou: %v, %v", ok, err)
	}

	ativo := true
	casos := []struct {
		nome     string
		filtro   portsout.FornecedorFiltro
		esperado []string
	}{
		{"todos em ordem de nome", portsout.FornecedorFiltro{}, []string{"Cerâmica Paraíba", "Ceramiche Italia", "Foshan Ceramics", "Guangzhou Tiles", "José Fretes"}},
		{"busca sem acento", portsout.FornecedorFiltro{Busca: "  CERAMICA "}, []string{"Cerâmica Paraíba"}},
		{"busca no meio do nome", portsout.FornecedorFiltro{Busca: "ceram"}, []string{"Cerâmica Paraíba", "Ceramiche Italia", "Foshan Ceramics"}},
		{"busca por documento com máscara", portsout.FornecedorFiltro{Busca: "529.982"}, []string{"José Fretes"}},
		{"busca por identificação estrangeira", portsout.FornecedorFiltro{Busca: "it01"}, []string{"Ceramiche Italia"}},
		{"busca curinga é texto", portsout.FornecedorFiltro{Busca: "%"}, []string{}},
		{"país", portsout.FornecedorFiltro{Pais: "CN"}, []string{"Foshan Ceramics", "Guangzhou Tiles"}},
		{"moeda", portsout.FornecedorFiltro{Moeda: domainfornecedor.MoedaReal}, []string{"Cerâmica Paraíba", "José Fretes"}},
		{"ativos da China", portsout.FornecedorFiltro{Pais: "CN", Ativo: &ativo}, []string{"Foshan Ceramics"}},
		{"segunda página", portsout.FornecedorFiltro{PaginacaoFiltro: domain.PaginacaoFiltro{Page: 2, Size: 2}}, []string{"Foshan Ceramics", "Guangzhou Tiles"}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			nomes := c.nomesListados(t, caso.filtro)
			if strings.Join(nomes, "|") != strings.Join(caso.esperado, "|") {
				t.Errorf("nomes = %v, esperado %v", nomes, caso.esperado)
			}
		})
	}
}

func TestFornecedorAtualizarAvancaAVersao(t *testing.T) {
	c := novoCenarioFornecedores(t)
	fornecedor := c.criar(t, c.nacional(t, "11.222.333/0001-81", "Cerâmica Paraíba"))

	err := fornecedor.Alterar(domainfornecedor.DadosFornecedor{
		Nome:        "Cerâmica Nordeste",
		MoedaPadrao: domainfornecedor.MoedaEuro,
		UF:          ufDe("PE"),
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}

	ok, err := c.fornecedores.Atualizar(c.ctx, fornecedor)
	if err != nil || !ok {
		t.Fatalf("Atualizar = %v, %v", ok, err)
	}
	if fornecedor.Versao != 2 {
		t.Errorf("versão em memória = %d, esperada 2", fornecedor.Versao)
	}

	salvo := c.obter(t, fornecedor.ID)
	if salvo.Versao != 2 || salvo.Nome != "Cerâmica Nordeste" || salvo.NomeBusca != "ceramica nordeste" || *salvo.UF != "PE" {
		t.Errorf("fornecedor salvo = %+v", salvo)
	}
	if salvo.MoedaPadrao != domainfornecedor.MoedaEuro || salvo.Documento.Valor() != "11222333000181" {
		t.Errorf("moeda %q e documento %q", salvo.MoedaPadrao, salvo.Documento.Valor())
	}
}

func TestFornecedorAtualizarComVersaoVelhaDaConflito(t *testing.T) {
	c := novoCenarioFornecedores(t)
	fornecedor := c.criar(t, c.nacional(t, "11.222.333/0001-81", "Cerâmica Paraíba"))

	copiaVelha := c.obter(t, fornecedor.ID)

	fornecedor.Desativar(time.Now().UTC())
	if ok, err := c.fornecedores.Atualizar(c.ctx, fornecedor); err != nil || !ok {
		t.Fatalf("primeira gravação = %v, %v", ok, err)
	}

	copiaVelha.Nome = "Nome perdido"
	ok, err := c.fornecedores.Atualizar(c.ctx, copiaVelha)
	if ok || !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "alterado por outra pessoa") {
		t.Errorf("gravação com versão velha = %v, %v; esperado conflito", ok, err)
	}
	if copiaVelha.Versao != 1 {
		t.Errorf("versão da cópia recusada mudou para %d", copiaVelha.Versao)
	}

	salvo := c.obter(t, fornecedor.ID)
	if salvo.Nome != "Cerâmica Paraíba" || salvo.Ativo || salvo.Versao != 2 {
		t.Errorf("a gravação velha não deveria ter efeito: %+v", salvo)
	}
}

func TestFornecedorAtualizarInexistenteNaoEncontra(t *testing.T) {
	c := novoCenarioFornecedores(t)
	fantasma := c.nacional(t, "11.222.333/0001-81", "Fantasma")

	if ok, err := c.fornecedores.Atualizar(c.ctx, fantasma); ok || err != nil {
		t.Errorf("Atualizar inexistente = %v, %v; esperado false sem erro", ok, err)
	}
}

func TestFornecedorEdicoesSimultaneasSoUmaGrava(t *testing.T) {
	c := novoCenarioFornecedores(t)
	fornecedor := c.criar(t, c.nacional(t, "11.222.333/0001-81", "Cerâmica Paraíba"))

	const concorrentes = 8
	var (
		espera     sync.WaitGroup
		largada    = make(chan struct{})
		resultados = make(chan error, concorrentes)
	)

	for indice := range concorrentes {
		copia := c.obter(t, fornecedor.ID)
		copia.Nome = "Edição " + string(rune('A'+indice))

		espera.Add(1)
		go func() {
			defer espera.Done()
			<-largada

			ok, err := c.fornecedores.Atualizar(c.ctx, copia)
			if err == nil && !ok {
				err = errors.New("fornecedor sumiu")
			}
			resultados <- err
		}()
	}

	close(largada)
	espera.Wait()
	close(resultados)

	gravadas, conflitos := 0, 0
	for err := range resultados {
		switch {
		case err == nil:
			gravadas++
		case errors.Is(err, domain.ErrConflito):
			conflitos++
		default:
			t.Errorf("erro inesperado: %v", err)
		}
	}
	if gravadas != 1 || conflitos != concorrentes-1 {
		t.Errorf("gravadas %d e conflitos %d; esperado 1 e %d", gravadas, conflitos, concorrentes-1)
	}
	if salvo := c.obter(t, fornecedor.ID); salvo.Versao != 2 {
		t.Errorf("versão final = %d, esperada 2", salvo.Versao)
	}
}

func TestFornecedorRemover(t *testing.T) {
	c := novoCenarioFornecedores(t)
	fornecedor := c.criar(t, c.estrangeiro(t, "CN", "", "Foshan", domainfornecedor.MoedaDolar))

	if ok, err := c.fornecedores.Remover(c.ctx, fornecedor.ID); err != nil || !ok {
		t.Fatalf("Remover = %v, %v", ok, err)
	}
	if ok, err := c.fornecedores.Remover(c.ctx, fornecedor.ID); err != nil || ok {
		t.Errorf("Remover de novo = %v, %v; esperado false", ok, err)
	}
	if removido, err := c.fornecedores.Obter(c.ctx, fornecedor.ID); err != nil || removido != nil {
		t.Errorf("Obter removido = %v, %v", removido, err)
	}
}
