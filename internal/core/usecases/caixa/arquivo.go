package caixa

import (
	"context"
	"fmt"
	"io"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

const (
	contentTypeOFX   = "application/x-ofx"
	tamanhoMaximoOFX = 5 << 20
)

type leitorDeOFX struct {
	arquivos portsout.ArquivoRepository
	storage  portsout.Storage
	extratos portsout.ExtratoRepository
}

func (l *leitorDeOFX) ler(ctx context.Context, arquivoID, usuarioID uuid.UUID) ([]byte, error) {
	if arquivoID == uuid.Nil {
		return nil, domain.ErroValidacao("arquivo OFX é obrigatório")
	}

	arquivo, err := l.arquivos.Obter(ctx, arquivoID)
	if err != nil {
		return nil, err
	}
	if arquivo == nil {
		return nil, domain.ErroNaoEncontrado("arquivo não encontrado")
	}
	if arquivo.ProprietarioID != usuarioID {
		return nil, domain.ErroPermissao("o arquivo OFX precisa ter sido enviado por você")
	}
	if arquivo.ContentType != contentTypeOFX {
		return nil, domain.ErroValidacao("arquivo deve ser um extrato OFX")
	}

	daSolicitacao, err := l.arquivos.VinculadoASolicitacao(ctx, arquivo.ID)
	if err != nil {
		return nil, err
	}
	if daSolicitacao {
		return nil, domain.ErroValidacao("arquivo de uma solicitação não pode ser importado como extrato")
	}

	importado, err := l.extratos.ArquivoImportado(ctx, arquivo.ID)
	if err != nil {
		return nil, err
	}
	if importado {
		return nil, domaincaixa.ErroArquivoJaImportado()
	}

	return l.conteudo(ctx, arquivo)
}

func (l *leitorDeOFX) conteudo(ctx context.Context, arquivo *domainsolicitacao.Arquivo) ([]byte, error) {
	leitor, err := l.storage.Baixar(ctx, arquivo.Chave)
	if err != nil {
		return nil, fmt.Errorf("baixar o OFX do armazenamento: %w", err)
	}
	defer func() { _ = leitor.Close() }()

	conteudo, err := io.ReadAll(io.LimitReader(leitor, tamanhoMaximoOFX+1))
	if err != nil {
		return nil, fmt.Errorf("ler o OFX do armazenamento: %w", err)
	}
	if len(conteudo) > tamanhoMaximoOFX {
		return nil, domain.ErroValidacao("extrato OFX deve ter no máximo 5MB")
	}

	return conteudo, nil
}
