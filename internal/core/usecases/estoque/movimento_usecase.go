package estoque

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.MovimentoUseCase = (*MovimentoUsecaseImpl)(nil)

type MovimentoUsecaseImpl struct {
	produtos   portsout.ProdutoRepository
	movimentos portsout.MovimentoRepository
	lotes      portsout.LoteRepository
	reservas   portsout.ReservaRepository
	operacoesDeEstoque
}

func NewMovimentoUsecase(
	produtos portsout.ProdutoRepository,
	movimentos portsout.MovimentoRepository,
	estoque portsout.EstoqueRepository,
	locais portsout.LocalRepository,
	lotes portsout.LoteRepository,
	reservas portsout.ReservaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *MovimentoUsecaseImpl {
	return &MovimentoUsecaseImpl{
		produtos:   produtos,
		movimentos: movimentos,
		lotes:      lotes,
		reservas:   reservas,
		operacoesDeEstoque: operacoesDeEstoque{
			estoque: estoque,
			locais:  locais,
			permissoes: permissoes{
				usuarios: usuarios,
				cargos:   cargos,
				empresas: empresas,
			},
		},
	}
}

func (u *MovimentoUsecaseImpl) Movimentar(ctx context.Context, input portsin.MovimentarEstoqueInput) (*domainestoque.Movimento, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	tipo, err := domainestoque.NovoTipoMovimento(input.Tipo)
	if err != nil {
		return nil, err
	}
	if tipo == domainestoque.TipoMovimentoTransferencia {
		return nil, domain.ErroValidacao("transferência usa a rota de transferências")
	}

	quantidade, err := domainestoque.NovaQuantidade(input.Quantidade)
	if err != nil {
		return nil, err
	}

	custo, err := custoInformado(input.CustoCentavos)
	if err != nil {
		return nil, err
	}

	local, err := u.localInformado(ctx, input.LocalID, "local do movimento não encontrado")
	if err != nil {
		return nil, err
	}
	if err := u.exigeAcessoAosLocais(ctx, input.UsuarioID, local); err != nil {
		return nil, err
	}

	reserva, err := u.reservaDaSaida(ctx, input.ReservaID)
	if err != nil {
		return nil, err
	}

	estoque, err := u.carregar(ctx, input.ProdutoID, local.ID)
	if err != nil {
		return nil, err
	}

	operacao, err := estoque.Movimentar(domainestoque.MovimentoLocalInput{
		Tipo:         tipo,
		Local:        local,
		Quantidade:   quantidade,
		Custo:        custo,
		Reserva:      reserva,
		UsuarioID:    input.UsuarioID,
		DocumentoRef: input.DocumentoRef,
		Observacao:   input.Observacao,
		Agora:        time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	if err := u.estoque.Gravar(ctx, operacao); err != nil {
		return nil, err
	}

	return operacao.Movimento, nil
}

func (u *MovimentoUsecaseImpl) Transferir(ctx context.Context, input portsin.TransferirEstoqueInput) (*domainestoque.Movimento, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	quantidade, err := domainestoque.NovaQuantidade(input.Quantidade)
	if err != nil {
		return nil, err
	}

	origem, err := u.localInformado(ctx, input.LocalOrigemID, "local de origem não encontrado")
	if err != nil {
		return nil, err
	}
	destino, err := u.localInformado(ctx, input.LocalDestinoID, "local de destino não encontrado")
	if err != nil {
		return nil, err
	}
	if err := u.exigeAcessoAosLocais(ctx, input.UsuarioID, origem, destino); err != nil {
		return nil, err
	}

	estoque, err := u.carregar(ctx, input.ProdutoID, origem.ID, destino.ID)
	if err != nil {
		return nil, err
	}

	operacao, err := estoque.Transferir(domainestoque.TransferenciaInput{
		Origem:       origem,
		Destino:      destino,
		Quantidade:   quantidade,
		UsuarioID:    input.UsuarioID,
		DocumentoRef: input.DocumentoRef,
		Observacao:   input.Observacao,
		Agora:        time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	if err := u.estoque.Gravar(ctx, operacao); err != nil {
		return nil, err
	}

	return operacao.Movimento, nil
}

func (u *MovimentoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarMovimentosInput) ([]*domainestoque.Movimento, error) {
	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, nil)
	if err != nil {
		return nil, err
	}

	filtro := portsout.MovimentoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		ProdutoID:       input.ProdutoID,
		EmpresaIDs:      empresaIDs,
	}

	if input.LocalID != nil {
		local, err := u.localDoFiltro(ctx, input.UsuarioID, *input.LocalID)
		if err != nil {
			return nil, err
		}
		filtro.LocalID = &local.ID
	}

	if input.Tipo != "" {
		tipo, err := domainestoque.NovoTipoMovimento(input.Tipo)
		if err != nil {
			return nil, err
		}
		filtro.Tipo = tipo
	}

	return u.movimentos.Listar(ctx, filtro)
}

func (u *MovimentoUsecaseImpl) Saldo(ctx context.Context, produtoID, usuarioID uuid.UUID) (*domainestoque.SaldoProduto, error) {
	empresaIDs, err := u.empresasVisiveis(ctx, usuarioID, nil)
	if err != nil {
		return nil, err
	}

	produto, err := u.produtos.Obter(ctx, produtoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, erroProdutoNaoEncontrado()
	}

	locais, err := u.estoque.SaldosDoProduto(ctx, produto.ID, empresaIDs)
	if err != nil {
		return nil, err
	}

	totalAlocado, err := u.estoque.TotalAlocado(ctx, produto.ID)
	if err != nil {
		return nil, err
	}

	return &domainestoque.SaldoProduto{
		Produto:    produto,
		NaoAlocado: produto.Saldo.Quantidade() - totalAlocado,
		Locais:     locais,
	}, nil
}

func (u *MovimentoUsecaseImpl) ListarLotes(ctx context.Context, input portsin.ListarLotesInput) ([]*domainestoque.Lote, error) {
	if err := u.exigeAdministradorOuFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, nil)
	if err != nil {
		return nil, err
	}

	produto, err := u.produtos.Obter(ctx, input.ProdutoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, erroProdutoNaoEncontrado()
	}

	filtro := portsout.LoteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		ProdutoID:       produto.ID,
		EmpresaIDs:      empresaIDs,
		ComSaldo:        input.ComSaldo,
	}

	if input.LocalID != nil {
		local, err := u.localDoFiltro(ctx, input.UsuarioID, *input.LocalID)
		if err != nil {
			return nil, err
		}
		filtro.LocalID = &local.ID
	}

	return u.lotes.Listar(ctx, filtro)
}

type operacoesDeEstoque struct {
	estoque portsout.EstoqueRepository
	locais  portsout.LocalRepository
	permissoes
}

func (u *MovimentoUsecaseImpl) reservaDaSaida(ctx context.Context, reservaID *uuid.UUID) (*domainestoque.Reserva, error) {
	if reservaID == nil {
		return nil, nil
	}

	reserva, err := u.reservas.Obter(ctx, *reservaID)
	if err != nil {
		return nil, err
	}
	if reserva == nil {
		return nil, domain.ErroValidacao("reserva não encontrada")
	}

	return reserva, nil
}

func (u *operacoesDeEstoque) carregar(ctx context.Context, produtoID uuid.UUID, localIDs ...uuid.UUID) (*domainestoque.EstoqueProduto, error) {
	estoque, err := u.estoque.Carregar(ctx, produtoID, localIDs)
	if err != nil {
		return nil, err
	}
	if estoque == nil {
		return nil, erroProdutoNaoEncontrado()
	}

	return estoque, nil
}

func (u *operacoesDeEstoque) localInformado(ctx context.Context, localID uuid.UUID, mensagem string) (*domainestoque.Local, error) {
	if localID == uuid.Nil {
		return nil, domain.ErroValidacao(mensagem)
	}

	local, err := u.locais.Obter(ctx, localID)
	if err != nil {
		return nil, err
	}
	if local == nil {
		return nil, domain.ErroValidacao(mensagem)
	}

	return local, nil
}

func (u *operacoesDeEstoque) localDoFiltro(ctx context.Context, usuarioID, localID uuid.UUID) (*domainestoque.Local, error) {
	local, err := u.localInformado(ctx, localID, "local do filtro não encontrado")
	if err != nil {
		return nil, err
	}
	if err := u.exigeAcessoAosLocais(ctx, usuarioID, local); err != nil {
		return nil, err
	}

	return local, nil
}

func custoInformado(centavos *int64) (*domainestoque.Custo, error) {
	if centavos == nil {
		return nil, nil
	}

	custo, err := domainestoque.NovoCusto(*centavos)
	if err != nil {
		return nil, err
	}

	return &custo, nil
}

func erroProdutoNaoEncontrado() error {
	return domain.ErroNaoEncontrado("produto não encontrado")
}
