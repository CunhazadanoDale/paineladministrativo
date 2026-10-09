package estoque

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoObservacaoMovimento = 1000

type Movimento struct {
	ID           uuid.UUID
	ProdutoID    uuid.UUID
	Tipo         TipoMovimento
	Quantidade   Quantidade
	SaldoApos    Saldo
	UsuarioID    uuid.UUID
	DocumentoRef *string
	Observacao   string
	CriadoEm     time.Time
}

func NovoMovimento(
	produtoID uuid.UUID,
	tipo TipoMovimento,
	quantidade Quantidade,
	saldoApos Saldo,
	usuarioID uuid.UUID,
	documentoRef *string,
	observacao string,
	agora time.Time,
) (*Movimento, error) {
	if produtoID == uuid.Nil {
		return nil, domain.ErroValidacao("produto da movimentação não informado")
	}
	if err := tipo.Validado(); err != nil {
		return nil, err
	}
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário da movimentação não informado")
	}
	if quantidade.Valor() == 0 {
		return nil, domain.ErroValidacao("quantidade da movimentação deve ser diferente de zero")
	}

	observacao = strings.TrimSpace(observacao)
	if utf8.RuneCountInString(observacao) > tamanhoMaximoObservacaoMovimento {
		return nil, domain.ErroValidacao("observação da movimentação deve ter no máximo 1000 caracteres")
	}

	return &Movimento{
		ID:           uuid.New(),
		ProdutoID:    produtoID,
		Tipo:         tipo,
		Quantidade:   quantidade,
		SaldoApos:    saldoApos,
		UsuarioID:    usuarioID,
		DocumentoRef: normalizarDocumentoRef(documentoRef),
		Observacao:   observacao,
		CriadoEm:     agora,
	}, nil
}

func normalizarDocumentoRef(documentoRef *string) *string {
	if documentoRef == nil {
		return nil
	}

	normalizado := strings.TrimSpace(*documentoRef)
	if normalizado == "" {
		return nil
	}

	return &normalizado
}
