package comissao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

const versaoInicial = 1

type PercentualPedido struct {
	PedidoID        uuid.UUID
	RepresentanteID uuid.UUID
	Percentual      domaincliente.Percentual
	Origem          OrigemPercentual
	UsuarioID       uuid.UUID
	Versao          int
	CriadoEm        time.Time
	AtualizadoEm    time.Time
}

func NovoPercentualManual(pedidoID, representanteID, usuarioID uuid.UUID, pontosBase int, agora time.Time) (*PercentualPedido, error) {
	if err := conferirIDs(pedidoID, representanteID, usuarioID); err != nil {
		return nil, err
	}

	percentual, err := domaincliente.NovoPercentual(pontosBase)
	if err != nil {
		return nil, err
	}

	return &PercentualPedido{
		PedidoID:        pedidoID,
		RepresentanteID: representanteID,
		Percentual:      percentual,
		Origem:          OrigemPercentualManual,
		UsuarioID:       usuarioID,
		Versao:          versaoInicial,
		CriadoEm:        agora,
		AtualizadoEm:    agora,
	}, nil
}

func PercentualPadrao(pedidoID uuid.UUID, modelo domainpedido.Modelo, representante *domaincliente.Representante, usuarioID uuid.UUID, agora time.Time) (*PercentualPedido, error) {
	if representante == nil {
		return nil, ErroRepresentanteNaoEncontrado()
	}
	if err := conferirIDs(pedidoID, representante.ID, usuarioID); err != nil {
		return nil, err
	}

	var percentual domaincliente.Percentual
	switch modelo {
	case domainpedido.ModeloProntaEntrega:
		percentual = representante.ComissaoProntaEntrega
	case domainpedido.ModeloImportacao:
		percentual = representante.ComissaoImportacao
	default:
		return nil, modelo.Validado()
	}

	return &PercentualPedido{
		PedidoID:        pedidoID,
		RepresentanteID: representante.ID,
		Percentual:      percentual,
		Origem:          OrigemPercentualPadrao,
		UsuarioID:       usuarioID,
		Versao:          versaoInicial,
		CriadoEm:        agora,
		AtualizadoEm:    agora,
	}, nil
}

func PercentualSugerido(pedidoID uuid.UUID, modelo domainpedido.Modelo, representante *domaincliente.Representante, usuarioID uuid.UUID, agora time.Time) (*PercentualPedido, error) {
	sugerido, err := PercentualPadrao(pedidoID, modelo, representante, usuarioID, agora)
	if err != nil {
		return nil, err
	}
	sugerido.Versao = 0

	return sugerido, nil
}

func (p *PercentualPedido) Gravado() bool {
	return p.Versao >= versaoInicial
}

func (p *PercentualPedido) VigentePara(representanteID uuid.UUID) bool {
	return p != nil && p.RepresentanteID == representanteID
}

func (p *PercentualPedido) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da comissão do pedido é obrigatória")
	}
	if informada != p.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (p *PercentualPedido) Redefinir(pontosBase int, usuarioID uuid.UUID, agora time.Time) error {
	if usuarioID == uuid.Nil {
		return domain.ErroValidacao("usuário não informado")
	}

	percentual, err := domaincliente.NovoPercentual(pontosBase)
	if err != nil {
		return err
	}

	p.Percentual = percentual
	p.Origem = OrigemPercentualManual
	p.UsuarioID = usuarioID
	p.AtualizadoEm = agora

	return nil
}

func conferirIDs(pedidoID, representanteID, usuarioID uuid.UUID) error {
	switch {
	case pedidoID == uuid.Nil:
		return domain.ErroValidacao("pedido da comissão não informado")
	case representanteID == uuid.Nil:
		return ErroPedidoSemRepresentante()
	case usuarioID == uuid.Nil:
		return domain.ErroValidacao("usuário não informado")
	default:
		return nil
	}
}
