package notafiscal

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

var _ portsin.NotaFiscalUseCase = (*NotaFiscalUsecaseImpl)(nil)

type NotaFiscalUsecaseImpl struct {
	produtos portsout.ProdutoRepository
	pedidos  portsout.PedidoRepository
	entregas portsout.EntregaRepository
	geradorDeTitulos
	leitorDeXML
	permissoes
}

func NewNotaFiscalUsecase(
	notas portsout.NotaFiscalRepository,
	arquivos portsout.ArquivoRepository,
	storage portsout.Storage,
	empresas portsout.EmpresaRepository,
	clientes portsout.ClienteRepository,
	produtos portsout.ProdutoRepository,
	pedidos portsout.PedidoRepository,
	entregas portsout.EntregaRepository,
	titulos portsout.TituloReceberRepository,
	representantes portsout.RepresentanteRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *NotaFiscalUsecaseImpl {
	return &NotaFiscalUsecaseImpl{
		produtos: produtos,
		pedidos:  pedidos,
		entregas: entregas,
		geradorDeTitulos: geradorDeTitulos{
			titulos: titulos,
		},
		leitorDeXML: leitorDeXML{
			arquivos: arquivos,
			storage:  storage,
			vinculos: notas,
		},
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
			clientes:       clientes,
			notas:          notas,
		},
	}
}

func (u *NotaFiscalUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarInput) (*portsin.Importacao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	conteudo, err := u.leitorDeXML.ler(ctx, input.ArquivoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	documento, err := domainnotafiscal.LerNFe(conteudo)
	if err != nil {
		return nil, err
	}

	emitente, err := u.empresaEmitente(ctx, perfil, documento)
	if err != nil {
		return nil, err
	}

	existente, err := u.notas.ObterPorChave(ctx, documento.Chave.Valor())
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, domain.ErroConflito("já existe uma nota fiscal com esta chave de acesso")
	}

	destino, err := u.destinoDaNota(ctx, documento)
	if err != nil {
		return nil, err
	}

	produtos, err := u.produtos.IDsPorCodigos(ctx, codigosDosItens(documento))
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	importacao := domainnotafiscal.ImportacaoInput{
		Documento:         documento,
		EmpresaID:         emitente.ID,
		ProdutosPorCodigo: produtos,
		ArquivoID:         input.ArquivoID,
		UsuarioID:         input.UsuarioID,
		Agora:             agora,
	}
	if destino.empresa != nil {
		importacao.EmpresaDestinatariaID = &destino.empresa.ID
	} else {
		importacao.ClienteID = &destino.cliente.ID
	}

	nota, err := domainnotafiscal.NovaNotaFiscal(importacao)
	if err != nil {
		return nil, err
	}

	if input.PedidoID != nil || input.EntregaID != nil {
		pedido, entrega, err := u.pedidoEEntrega(ctx, input.PedidoID, input.EntregaID)
		if err != nil {
			return nil, err
		}
		if err := nota.Vincular(pedido, entrega, agora); err != nil {
			return nil, err
		}
	}

	if destino.clienteNovo {
		if err := u.gravarClienteNovo(ctx, destino, nota); err != nil {
			return nil, err
		}
	}

	if err := u.notas.Criar(ctx, nota); err != nil {
		return nil, err
	}

	importada := &portsin.Importacao{Nota: nota, Avisos: avisosDaImportacao(nota, destino), TitulosGerados: []uuid.UUID{}}
	if nota.GeraTitulos() {
		resultado, err := u.gerar(ctx, nota, input.UsuarioID, agora)
		if err != nil {
			importada.ErroDosTitulos = err
			importada.Avisos = append(importada.Avisos, portsin.AvisoTitulosPendentes)
		} else {
			importada.TitulosGerados = resultado.Gerados
		}
	}

	return importada, nil
}

func (u *NotaFiscalUsecaseImpl) SincronizarTitulos(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) (*portsin.SincronizacaoTitulos, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	nota, err := u.notaVisivel(ctx, perfil, notaFiscalID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	sincronizacao := &portsin.SincronizacaoTitulos{
		Gerados: []uuid.UUID{}, JaExistentes: []uuid.UUID{}, Cancelados: []uuid.UUID{}, Marcados: []uuid.UUID{},
	}

	switch {
	case nota.Status == domainnotafiscal.StatusCancelada:
		if !nota.PodeTerTitulos() {
			return sincronizacao, nil
		}
		resultado, err := u.cancelar(ctx, nota, usuarioID, agora)
		if err != nil {
			return nil, err
		}
		sincronizacao.Cancelados, sincronizacao.Marcados, sincronizacao.Ignorados = resultado.Cancelados, resultado.Marcados, resultado.Ignorados
	case nota.GeraTitulos():
		resultado, err := u.gerar(ctx, nota, usuarioID, agora)
		if err != nil {
			return nil, err
		}
		sincronizacao.Gerados, sincronizacao.JaExistentes = resultado.Gerados, resultado.JaExistentes
	default:
		return nil, domain.ErroValidacao("só nota fiscal de venda, autorizada e com duplicatas gera títulos")
	}

	return sincronizacao, nil
}

func (u *NotaFiscalUsecaseImpl) Obter(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) (*domainnotafiscal.NotaFiscal, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	return u.notaVisivel(ctx, perfil, notaFiscalID)
}

func (u *NotaFiscalUsecaseImpl) Listar(ctx context.Context, input portsin.ListarNotasInput) ([]*domainnotafiscal.NotaFiscal, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	filtro := portsout.NotaFiscalFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		ClienteID:       input.ClienteID,
		RepresentanteID: escopo.representanteID,
		PedidoID:        input.PedidoID,
		EntreEmpresas:   input.EntreEmpresas,
		SemPedido:       input.SemPedido,
		Numero:          input.Numero,
		Chave:           input.Chave,
	}
	if input.Natureza != "" {
		if filtro.Natureza, err = domainnotafiscal.NovaNatureza(input.Natureza); err != nil {
			return nil, err
		}
	}
	if input.Status != "" {
		if filtro.Status, err = domainnotafiscal.NovoStatus(input.Status); err != nil {
			return nil, err
		}
	}
	if filtro.EmitidaDe, filtro.EmitidaAte, err = periodo(input.EmitidaDe, input.EmitidaAte); err != nil {
		return nil, err
	}

	return u.notas.Listar(ctx, filtro)
}

func (u *NotaFiscalUsecaseImpl) Vincular(ctx context.Context, input portsin.VincularInput) (*domainnotafiscal.NotaFiscal, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	nota, err := u.notaVisivel(ctx, perfil, input.NotaFiscalID)
	if err != nil {
		return nil, err
	}
	if err := nota.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	var pedido *domainpedido.Pedido
	var entrega *domainpedido.Entrega
	if input.PedidoID != nil || input.EntregaID != nil {
		if pedido, entrega, err = u.pedidoEEntrega(ctx, input.PedidoID, input.EntregaID); err != nil {
			return nil, err
		}
	}

	versaoLida, statusAnterior := nota.Versao, nota.Status
	if err := nota.Vincular(pedido, entrega, time.Now().UTC()); err != nil {
		return nil, err
	}

	if err := u.notas.Atualizar(ctx, &portsout.Alteracao{
		Nota:            nota,
		VersaoLida:      versaoLida,
		StatusAnterior:  statusAnterior,
		ConferirVinculo: true,
	}); err != nil {
		return nil, err
	}

	return nota, nil
}

func (u *NotaFiscalUsecaseImpl) VincularProduto(ctx context.Context, input portsin.VincularProdutoInput) (*domainnotafiscal.NotaFiscal, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	nota, err := u.notaVisivel(ctx, perfil, input.NotaFiscalID)
	if err != nil {
		return nil, err
	}
	if err := nota.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	if input.ProdutoID == uuid.Nil {
		return nil, domain.ErroValidacao("produto é obrigatório")
	}
	produto, err := u.produtos.Obter(ctx, input.ProdutoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, domain.ErroNaoEncontrado("produto não encontrado")
	}

	versaoLida, statusAnterior := nota.Versao, nota.Status
	item, err := nota.VincularProduto(input.NumeroItem, produto.ID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := u.notas.Atualizar(ctx, &portsout.Alteracao{
		Nota:           nota,
		VersaoLida:     versaoLida,
		StatusAnterior: statusAnterior,
		ItensAlterados: []*domainnotafiscal.Item{item},
	}); err != nil {
		return nil, err
	}

	return nota, nil
}

func (u *NotaFiscalUsecaseImpl) BaixarXML(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, nil, err
	}

	nota, err := u.notaVisivel(ctx, perfil, notaFiscalID)
	if err != nil {
		return nil, nil, err
	}

	arquivo, err := u.arquivos.Obter(ctx, nota.ArquivoID)
	if err != nil {
		return nil, nil, err
	}
	if arquivo == nil {
		return nil, nil, domain.ErroNaoEncontrado("arquivo XML da nota não encontrado")
	}

	conteudo, err := u.storage.Baixar(ctx, arquivo.Chave)
	if err != nil {
		return nil, nil, err
	}

	return arquivo, conteudo, nil
}

func (u *NotaFiscalUsecaseImpl) ListarItensSemProduto(ctx context.Context, input portsin.ListarItensSemProdutoInput) ([]*domainnotafiscal.ItemPendente, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if perfil.restritoAoRepresentante() {
		return nil, domain.ErroPermissao("perfil sem permissão para consultar itens sem produto")
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.notas.ListarItensSemProduto(ctx, portsout.ItemPendenteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
	})
}

func (u *NotaFiscalUsecaseImpl) ListarEntregasSemNota(ctx context.Context, input portsin.ListarEntregasSemNotaInput) ([]*domainnotafiscal.EntregaSemNota, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.notas.ListarEntregasSemNota(ctx, portsout.EntregaSemNotaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		ClienteID:       input.ClienteID,
		RepresentanteID: escopo.representanteID,
	})
}

func (u *NotaFiscalUsecaseImpl) empresaEmitente(ctx context.Context, perfil perfilDoUsuario, documento *domainnotafiscal.Documento) (*domainempresa.Empresa, error) {
	emitente, err := u.empresas.ObterPorCNPJ(ctx, domainempresa.CNPJDe(documento.EmitenteCNPJ))
	if err != nil {
		return nil, err
	}
	if emitente == nil {
		return nil, domain.ErroValidacao("o emitente da nota não é uma empresa cadastrada")
	}
	if !perfil.acesso.Permite(emitente.ID) {
		return nil, erroSemAcessoAEmpresa()
	}
	if !emitente.Ativa {
		return nil, domain.ErroValidacao("a empresa emitente está inativa")
	}

	return emitente, nil
}

func (u *NotaFiscalUsecaseImpl) pedidoEEntrega(ctx context.Context, pedidoID, entregaID *uuid.UUID) (*domainpedido.Pedido, *domainpedido.Entrega, error) {
	if pedidoID == nil {
		return nil, nil, domain.ErroValidacao("entrega exige o pedido")
	}

	pedido, err := u.pedidos.Obter(ctx, *pedidoID)
	if err != nil {
		return nil, nil, err
	}
	if pedido == nil {
		return nil, nil, domain.ErroNaoEncontrado("pedido não encontrado")
	}

	if entregaID == nil {
		return pedido, nil, nil
	}

	entrega, err := u.entregas.Obter(ctx, *entregaID)
	if err != nil {
		return nil, nil, err
	}
	if entrega == nil {
		return nil, nil, domain.ErroNaoEncontrado("entrega não encontrada")
	}

	return pedido, entrega, nil
}

func (u *NotaFiscalUsecaseImpl) gravarClienteNovo(ctx context.Context, destino *destinoDaNota, nota *domainnotafiscal.NotaFiscal) error {
	_, err := u.clientes.Criar(ctx, destino.cliente)
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrConflito) {
		return err
	}

	existente, erroLeitura := u.clientes.ObterPorDocumento(ctx, destino.cliente.Documento.Valor())
	if erroLeitura != nil {
		return erroLeitura
	}
	if existente == nil {
		return err
	}

	destino.cliente = existente
	destino.clienteNovo = false
	nota.ClienteID = &existente.ID

	return nil
}

func avisosDaImportacao(nota *domainnotafiscal.NotaFiscal, destino *destinoDaNota) []string {
	avisos := make([]string, 0, 3)
	if destino.cliente != nil && nota.FaltaNumeroPedido(destino.cliente.ExigeNumeroPedidoNaNF) {
		avisos = append(avisos, portsin.AvisoNumeroPedidoAusente)
	}
	if len(nota.ItensSemProduto()) > 0 {
		avisos = append(avisos, portsin.AvisoItensSemProduto)
	}
	if destino.clienteNovo {
		avisos = append(avisos, portsin.AvisoClienteCriado)
	}

	return avisos
}

func codigosDosItens(documento *domainnotafiscal.Documento) []string {
	vistos := make(map[string]bool, len(documento.Itens))
	codigos := make([]string, 0, len(documento.Itens))
	for _, item := range documento.Itens {
		if !vistos[item.Codigo] {
			vistos[item.Codigo] = true
			codigos = append(codigos, item.Codigo)
		}
	}

	return codigos
}

func periodo(de, ate *time.Time) (*time.Time, *time.Time, error) {
	var inicio, fim *time.Time
	if de != nil {
		data := domainpedido.DataDe(*de)
		inicio = &data
	}
	if ate != nil {
		data := domainpedido.DataDe(*ate)
		fim = &data
	}
	if inicio != nil && fim != nil && fim.Before(*inicio) {
		return nil, nil, domain.ErroValidacao("data final do período antes da inicial")
	}

	return inicio, fim, nil
}
