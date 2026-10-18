package comissao

import (
	"math"
	"math/big"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

const pontosBaseDoTodo = 10000

type Comissao struct {
	ID              uuid.UUID
	EmpresaID       uuid.UUID
	RepresentanteID uuid.UUID
	PedidoID        uuid.UUID
	EntregaID       uuid.UUID
	Competencia     Competencia
	BaseCentavos    int64
	Percentual      domaincliente.Percentual
	ValorCentavos   int64
	FechamentoID    *uuid.UUID
	UsuarioID       uuid.UUID
	CriadoEm        time.Time
}

type ItemSaido struct {
	Quantidade            int
	PrecoUnitarioCentavos int64
}

type EntregaParaApurar struct {
	EntregaID       uuid.UUID
	PedidoID        uuid.UUID
	PedidoNumero    int
	EmpresaID       uuid.UUID
	RepresentanteID uuid.UUID
	Modelo          domainpedido.Modelo
	SaiuEm          time.Time
	Percentual      *PercentualPedido
	Itens           []ItemSaido
}

type Apuracao struct {
	Comissao       *Comissao
	Percentual     *PercentualPedido
	PercentualNovo bool
}

type FalhaApuracao struct {
	EntregaID uuid.UUID
	Motivo    string
}

type ResultadoApuracao struct {
	Apuradas       []uuid.UUID
	JaExistentes   int
	Falhas         []FalhaApuracao
	RestamEntregas bool
}

func NovaApuracao(entrega EntregaParaApurar, representante *domaincliente.Representante, usuarioID uuid.UUID, agora time.Time) (*Apuracao, error) {
	if entrega.EntregaID == uuid.Nil || entrega.PedidoID == uuid.Nil || entrega.EmpresaID == uuid.Nil {
		return nil, domain.ErroValidacao("entrega da comissão não informada")
	}
	if entrega.RepresentanteID == uuid.Nil {
		return nil, ErroPedidoSemRepresentante()
	}
	if entrega.SaiuEm.IsZero() {
		return nil, domain.ErroValidacao("entrega sem data de saída não gera comissão")
	}
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}

	apuracao := &Apuracao{Percentual: entrega.Percentual}
	if !entrega.Percentual.VigentePara(entrega.RepresentanteID) {
		if representante == nil || representante.ID != entrega.RepresentanteID {
			return nil, ErroRepresentanteNaoEncontrado()
		}
		padrao, err := PercentualPadrao(entrega.PedidoID, entrega.Modelo, representante, usuarioID, agora)
		if err != nil {
			return nil, err
		}
		apuracao.Percentual = padrao
		apuracao.PercentualNovo = true
	}

	base, err := baseDaEntrega(entrega.Itens)
	if err != nil {
		return nil, err
	}

	apuracao.Comissao = &Comissao{
		ID:              uuid.New(),
		EmpresaID:       entrega.EmpresaID,
		RepresentanteID: entrega.RepresentanteID,
		PedidoID:        entrega.PedidoID,
		EntregaID:       entrega.EntregaID,
		Competencia:     CompetenciaDe(entrega.SaiuEm),
		BaseCentavos:    base,
		Percentual:      apuracao.Percentual.Percentual,
		ValorCentavos:   valorDaComissao(base, apuracao.Percentual.Percentual),
		UsuarioID:       usuarioID,
		CriadoEm:        agora,
	}

	return apuracao, nil
}

func baseDaEntrega(itens []ItemSaido) (int64, error) {
	if len(itens) == 0 {
		return 0, domain.ErroValidacao("entrega sem itens não gera comissão")
	}

	var base int64
	for _, item := range itens {
		if item.Quantidade <= 0 || item.PrecoUnitarioCentavos <= 0 {
			return 0, domain.ErroValidacao("item da entrega com quantidade ou preço inválido")
		}
		if item.PrecoUnitarioCentavos > math.MaxInt64/int64(item.Quantidade) {
			return 0, erroBaseForaDoLimite()
		}
		subtotal := item.PrecoUnitarioCentavos * int64(item.Quantidade)
		if base > math.MaxInt64-subtotal {
			return 0, erroBaseForaDoLimite()
		}
		base += subtotal
	}

	return base, nil
}

func valorDaComissao(base int64, percentual domaincliente.Percentual) int64 {
	produto := new(big.Int).Mul(big.NewInt(base), big.NewInt(int64(percentual.PontosBase())))
	produto.Add(produto, big.NewInt(pontosBaseDoTodo/2))

	return produto.Quo(produto, big.NewInt(pontosBaseDoTodo)).Int64()
}

func erroBaseForaDoLimite() error {
	return domain.ErroValidacao("valor da entrega excede o limite da comissão")
}
