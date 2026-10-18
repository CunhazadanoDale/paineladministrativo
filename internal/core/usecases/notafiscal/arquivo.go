package notafiscal

import (
	"context"
	"fmt"
	"io"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

const (
	contentTypeXML   = "application/xml"
	tamanhoMaximoXML = 10 << 20
)

type leitorDeXML struct {
	arquivos portsout.ArquivoRepository
	storage  portsout.Storage
	vinculos portsout.NotaFiscalRepository
}

func (l *leitorDeXML) ler(ctx context.Context, arquivoID, usuarioID uuid.UUID) ([]byte, error) {
	if arquivoID == uuid.Nil {
		return nil, domain.ErroValidacao("arquivo XML é obrigatório")
	}

	arquivo, err := l.arquivos.Obter(ctx, arquivoID)
	if err != nil {
		return nil, err
	}
	if arquivo == nil {
		return nil, domain.ErroNaoEncontrado("arquivo não encontrado")
	}
	if arquivo.ProprietarioID != usuarioID {
		return nil, domain.ErroPermissao("o arquivo XML precisa ter sido enviado por você")
	}
	if arquivo.ContentType != contentTypeXML {
		return nil, domain.ErroValidacao("arquivo deve ser um XML")
	}

	daSolicitacao, err := l.arquivos.VinculadoASolicitacao(ctx, arquivo.ID)
	if err != nil {
		return nil, err
	}
	if daSolicitacao {
		return nil, domain.ErroValidacao("arquivo de uma solicitação não pode ser importado como nota fiscal")
	}

	importado, err := l.vinculos.ArquivoVinculado(ctx, arquivo.ID)
	if err != nil {
		return nil, err
	}
	if importado {
		return nil, domain.ErroConflito("arquivo já importado em outra nota fiscal ou evento")
	}

	return l.conteudo(ctx, arquivo)
}

func (l *leitorDeXML) conteudo(ctx context.Context, arquivo *domainsolicitacao.Arquivo) ([]byte, error) {
	leitor, err := l.storage.Baixar(ctx, arquivo.Chave)
	if err != nil {
		return nil, fmt.Errorf("baixar o XML do armazenamento: %w", err)
	}
	defer func() { _ = leitor.Close() }()

	conteudo, err := io.ReadAll(io.LimitReader(leitor, tamanhoMaximoXML+1))
	if err != nil {
		return nil, fmt.Errorf("ler o XML do armazenamento: %w", err)
	}
	if len(conteudo) > tamanhoMaximoXML {
		return nil, domain.ErroValidacao("XML deve ter no máximo 10MB")
	}

	return conteudo, nil
}
