package fornecedor

import (
	"context"
	"strconv"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
)

var _ portsin.ImportacaoUseCase = (*FornecedorUsecaseImpl)(nil)

type importacaoDeFornecedores struct {
	simular        bool
	documentos     map[string]int
	identificacoes map[string]int
}

func (u *FornecedorUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarFornecedoresInput) (*domain.RelatorioImportacao, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}
	relatorio, err := domain.NovoRelatorioImportacao(input.Simular, len(input.Linhas))
	if err != nil {
		return nil, err
	}

	estado := &importacaoDeFornecedores{
		simular:        input.Simular,
		documentos:     map[string]int{},
		identificacoes: map[string]int{},
	}
	for _, linha := range input.Linhas {
		motivo, err := u.importarLinha(ctx, estado, linha)
		if err != nil {
			return nil, err
		}
		if motivo != "" {
			relatorio.Rejeitar(linha.Numero, chaveDaLinha(linha), motivo)
			continue
		}
		relatorio.Importadas++
	}

	return relatorio, nil
}

func (u *FornecedorUsecaseImpl) importarLinha(ctx context.Context, estado *importacaoDeFornecedores, linha portsin.LinhaImportacao) (string, error) {
	pais, err := domainfornecedor.NovoPais(linha.Pais)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	var documento domaincliente.Documento
	if strings.TrimSpace(linha.Documento) != "" {
		if documento, err = domaincliente.NovoDocumento(linha.Documento); err != nil {
			return domain.MotivoDeRejeicao(err)
		}
	}

	dados, err := dadosDoFornecedor(portsin.DadosFornecedorInput{
		IdentificacaoEstrangeira: opcional(linha.IdentificacaoEstrangeira),
		Nome:                     linha.Nome,
		NomeFantasia:             opcional(linha.NomeFantasia),
		MoedaPadrao:              linha.MoedaPadrao,
		UF:                       opcional(linha.UF),
		Cidade:                   opcional(linha.Cidade),
		ContatoNome:              opcional(linha.ContatoNome),
		Email:                    opcional(linha.Email),
		Telefone:                 opcional(linha.Telefone),
		Observacoes:              linha.Observacoes,
	})
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	fornecedor, err := domainfornecedor.NovoFornecedor(pais, documento, dados)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	chaveDocumento := fornecedor.Documento.Valor()
	if !fornecedor.Documento.Vazio() {
		if anterior, repetido := estado.documentos[chaveDocumento]; repetido {
			return "documento repetido no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
		}
	}
	chaveIdentificacao := ""
	if fornecedor.IdentificacaoEstrangeira != nil {
		chaveIdentificacao = string(fornecedor.Pais) + "|" + *fornecedor.IdentificacaoEstrangeira
		if anterior, repetido := estado.identificacoes[chaveIdentificacao]; repetido {
			return "identificação estrangeira repetida no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
		}
	}

	if err := u.conferirDocumentoLivre(ctx, fornecedor); err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	if err := u.conferirIdentificacaoLivre(ctx, fornecedor); err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	if !estado.simular {
		if _, err := u.fornecedores.Criar(ctx, fornecedor); err != nil {
			return domain.MotivoDeRejeicao(err)
		}
	}

	if !fornecedor.Documento.Vazio() {
		estado.documentos[chaveDocumento] = linha.Numero
	}
	if chaveIdentificacao != "" {
		estado.identificacoes[chaveIdentificacao] = linha.Numero
	}

	return "", nil
}

func chaveDaLinha(linha portsin.LinhaImportacao) string {
	for _, valor := range []string{linha.Documento, linha.IdentificacaoEstrangeira, linha.Nome} {
		if limpo := strings.TrimSpace(valor); limpo != "" {
			return limpo
		}
	}

	return ""
}

func opcional(valor string) *string {
	limpo := strings.TrimSpace(valor)
	if limpo == "" {
		return nil
	}

	return &limpo
}
