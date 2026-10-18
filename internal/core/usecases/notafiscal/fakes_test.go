package notafiscal_test

import (
	"bytes"
	"context"
	"io"
	"sort"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios          map[uuid.UUID]*domainusuarios.Usuario
	cargos            map[uuid.UUID]*domainusuarios.Cargo
	empresas          map[uuid.UUID]*domainempresa.Empresa
	acessos           map[uuid.UUID]domainempresa.Acesso
	representantes    map[uuid.UUID]*domaincliente.Representante
	clientes          map[uuid.UUID]*domaincliente.Cliente
	produtos          map[uuid.UUID]*domainestoque.Produto
	pedidos           map[uuid.UUID]*domainpedido.Pedido
	entregas          map[uuid.UUID]*domainpedido.Entrega
	arquivos          map[uuid.UUID]*domainsolicitacao.Arquivo
	conteudos         map[string][]byte
	daSolicitacao     map[uuid.UUID]bool
	notas             map[uuid.UUID]*domainnotafiscal.NotaFiscal
	eventos           []*domainnotafiscal.Evento
	filtroNotas       portsout.NotaFiscalFiltro
	filtroPendentes   portsout.ItemPendenteFiltro
	filtroEntregas    portsout.EntregaSemNotaFiltro
	conflitoNoCliente bool
	geracoes          []*domainreceber.Geracao
	cancelamentos     []domainreceber.CancelamentoDeOrigem
	titulosPorOrigem  map[uuid.UUID]uuid.UUID
	erroDosTitulos    error
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:         map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:           map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:         map[uuid.UUID]*domainempresa.Empresa{},
		acessos:          map[uuid.UUID]domainempresa.Acesso{},
		representantes:   map[uuid.UUID]*domaincliente.Representante{},
		clientes:         map[uuid.UUID]*domaincliente.Cliente{},
		produtos:         map[uuid.UUID]*domainestoque.Produto{},
		pedidos:          map[uuid.UUID]*domainpedido.Pedido{},
		entregas:         map[uuid.UUID]*domainpedido.Entrega{},
		arquivos:         map[uuid.UUID]*domainsolicitacao.Arquivo{},
		conteudos:        map[string][]byte{},
		daSolicitacao:    map[uuid.UUID]bool{},
		notas:            map[uuid.UUID]*domainnotafiscal.NotaFiscal{},
		titulosPorOrigem: map[uuid.UUID]uuid.UUID{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

func copiarNota(nota *domainnotafiscal.NotaFiscal) *domainnotafiscal.NotaFiscal {
	if nota == nil {
		return nil
	}

	copia := *nota
	copia.Itens = make([]*domainnotafiscal.Item, 0, len(nota.Itens))
	for _, item := range nota.Itens {
		copia.Itens = append(copia.Itens, copiar(item))
	}
	copia.Duplicatas = make([]*domainnotafiscal.Duplicata, 0, len(nota.Duplicatas))
	for _, duplicata := range nota.Duplicatas {
		copia.Duplicatas = append(copia.Duplicatas, copiar(duplicata))
	}

	return &copia
}

type repoUsuarios struct{ *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return copiar(r.usuarios[id]), nil
}

type repoCargos struct{ *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return copiar(r.cargos[id]), nil
}

type repoEmpresas struct{ *memoria }

func (r repoEmpresas) ObterPorCNPJ(_ context.Context, cnpj domainempresa.CNPJ) (*domainempresa.Empresa, error) {
	for _, empresa := range r.empresas {
		if empresa.CNPJ.Digitos() == cnpj.Digitos() {
			return copiar(empresa), nil
		}
	}

	return nil, nil
}

func (r repoEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return r.acessos[usuarioID], nil
}

type repoRepresentantes struct{ *memoria }

func (r repoRepresentantes) ObterPorUsuario(_ context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	for _, representante := range r.representantes {
		if representante.UsuarioID != nil && *representante.UsuarioID == usuarioID {
			return copiar(representante), nil
		}
	}

	return nil, nil
}

type repoClientes struct{ *memoria }

func (r repoClientes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Cliente, error) {
	return copiar(r.clientes[id]), nil
}

func (r repoClientes) ObterPorDocumento(_ context.Context, documento string) (*domaincliente.Cliente, error) {
	for _, cliente := range r.clientes {
		if cliente.Documento.Valor() == documento {
			return copiar(cliente), nil
		}
	}

	return nil, nil
}

func (r repoClientes) Criar(_ context.Context, cliente *domaincliente.Cliente) (uuid.UUID, error) {
	if r.conflitoNoCliente {
		concorrente := copiar(cliente)
		concorrente.ID = uuid.New()
		r.clientes[concorrente.ID] = concorrente
		return uuid.Nil, domain.ErroConflito("já existe um cliente com este documento")
	}
	for _, existente := range r.clientes {
		if existente.Documento.Valor() == cliente.Documento.Valor() {
			return uuid.Nil, domain.ErroConflito("já existe um cliente com este documento")
		}
	}
	r.clientes[cliente.ID] = copiar(cliente)

	return cliente.ID, nil
}

type repoProdutos struct{ *memoria }

func (r repoProdutos) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	return copiar(r.produtos[id]), nil
}

func (r repoProdutos) IDsPorCodigos(_ context.Context, codigos []string) (map[string]uuid.UUID, error) {
	encontrados := map[string]uuid.UUID{}
	for _, produto := range r.produtos {
		for _, codigo := range codigos {
			if produto.Codigo != nil && *produto.Codigo == codigo {
				encontrados[codigo] = produto.ID
			}
		}
	}

	return encontrados, nil
}

type repoPedidos struct{ *memoria }

func (r repoPedidos) Obter(_ context.Context, id uuid.UUID) (*domainpedido.Pedido, error) {
	return copiar(r.pedidos[id]), nil
}

type repoEntregas struct{ *memoria }

func (r repoEntregas) Obter(_ context.Context, id uuid.UUID) (*domainpedido.Entrega, error) {
	return copiar(r.entregas[id]), nil
}

type repoArquivos struct{ *memoria }

func (r repoArquivos) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error) {
	return copiar(r.arquivos[id]), nil
}

func (r repoArquivos) VinculadoASolicitacao(_ context.Context, arquivoID uuid.UUID) (bool, error) {
	return r.daSolicitacao[arquivoID], nil
}

type storageFalso struct{ *memoria }

func (s storageFalso) Baixar(_ context.Context, chave string) (io.ReadCloser, error) {
	conteudo, ok := s.conteudos[chave]
	if !ok {
		return nil, domain.ErroNaoEncontrado("objeto não encontrado no armazenamento")
	}

	return io.NopCloser(bytes.NewReader(conteudo)), nil
}

type repoNotas struct{ *memoria }

func (r repoNotas) Criar(_ context.Context, nota *domainnotafiscal.NotaFiscal) error {
	for _, existente := range r.notas {
		if existente.Chave.Valor() == nota.Chave.Valor() {
			return domain.ErroConflito("já existe uma nota fiscal com esta chave de acesso")
		}
		if existente.ArquivoID == nota.ArquivoID {
			return domain.ErroConflito("arquivo já importado em outra nota fiscal ou evento")
		}
	}
	r.notas[nota.ID] = copiarNota(nota)

	return nil
}

func (r repoNotas) Obter(_ context.Context, id uuid.UUID) (*domainnotafiscal.NotaFiscal, error) {
	return copiarNota(r.notas[id]), nil
}

func (r repoNotas) ObterPorChave(_ context.Context, chave string) (*domainnotafiscal.NotaFiscal, error) {
	for _, nota := range r.notas {
		if nota.Chave.Valor() == chave {
			return copiarNota(nota), nil
		}
	}

	return nil, nil
}

func (r repoNotas) Listar(_ context.Context, filtro portsout.NotaFiscalFiltro) ([]*domainnotafiscal.NotaFiscal, error) {
	r.filtroNotas = filtro
	permitidas := map[uuid.UUID]bool{}
	for _, id := range filtro.EmpresaIDs {
		permitidas[id] = true
	}

	notas := make([]*domainnotafiscal.NotaFiscal, 0)
	for _, nota := range r.notas {
		if filtro.EmpresaIDs != nil && !permitidas[nota.EmpresaID] {
			continue
		}
		if filtro.Empresa != nil && nota.EmpresaID != *filtro.Empresa {
			continue
		}
		notas = append(notas, copiarNota(nota))
	}
	sort.Slice(notas, func(i, j int) bool { return notas[i].Numero < notas[j].Numero })

	return notas, nil
}

func (r repoNotas) Atualizar(_ context.Context, alteracao *portsout.Alteracao) error {
	atual, ok := r.notas[alteracao.Nota.ID]
	if !ok {
		return domain.ErroNaoEncontrado("nota fiscal não encontrada")
	}
	if atual.Versao != alteracao.VersaoLida || atual.Status != alteracao.StatusAnterior {
		return domainnotafiscal.ErroVersaoDivergente()
	}
	if evento := alteracao.Evento; evento != nil {
		for _, existente := range r.eventos {
			if existente.NotaFiscalID == evento.NotaFiscalID && existente.Tipo == evento.Tipo && existente.Sequencia == evento.Sequencia {
				return domain.ErroConflito("evento já registrado nesta nota fiscal")
			}
		}
		r.eventos = append(r.eventos, copiar(evento))
	}

	alteracao.Nota.Versao = atual.Versao + 1
	r.notas[alteracao.Nota.ID] = copiarNota(alteracao.Nota)

	return nil
}

func (r repoNotas) ListarEventos(_ context.Context, notaFiscalID uuid.UUID) ([]*domainnotafiscal.Evento, error) {
	eventos := make([]*domainnotafiscal.Evento, 0)
	for _, evento := range r.eventos {
		if evento.NotaFiscalID == notaFiscalID {
			eventos = append(eventos, copiar(evento))
		}
	}

	return eventos, nil
}

func (r repoNotas) ListarItensSemProduto(_ context.Context, filtro portsout.ItemPendenteFiltro) ([]*domainnotafiscal.ItemPendente, error) {
	r.filtroPendentes = filtro

	return []*domainnotafiscal.ItemPendente{}, nil
}

func (r repoNotas) ListarEntregasSemNota(_ context.Context, filtro portsout.EntregaSemNotaFiltro) ([]*domainnotafiscal.EntregaSemNota, error) {
	r.filtroEntregas = filtro

	return []*domainnotafiscal.EntregaSemNota{}, nil
}

func (r repoNotas) ArquivoVinculado(_ context.Context, arquivoID uuid.UUID) (bool, error) {
	for _, nota := range r.notas {
		if nota.ArquivoID == arquivoID {
			return true, nil
		}
	}
	for _, evento := range r.eventos {
		if evento.ArquivoID == arquivoID {
			return true, nil
		}
	}

	return false, nil
}

type repoTitulos struct{ *memoria }

func (r repoTitulos) GerarDeOrigem(_ context.Context, geracao *domainreceber.Geracao) (domainreceber.ResultadoGeracao, error) {
	if r.erroDosTitulos != nil {
		return domainreceber.ResultadoGeracao{}, r.erroDosTitulos
	}
	r.geracoes = append(r.geracoes, geracao)

	resultado := domainreceber.ResultadoGeracao{Gerados: []uuid.UUID{}, JaExistentes: []uuid.UUID{}}
	for _, titulo := range geracao.Titulos {
		if existente, ok := r.titulosPorOrigem[*titulo.OrigemID]; ok {
			resultado.JaExistentes = append(resultado.JaExistentes, existente)
			continue
		}
		r.titulosPorOrigem[*titulo.OrigemID] = titulo.ID
		resultado.Gerados = append(resultado.Gerados, titulo.ID)
	}

	return resultado, nil
}

func (r repoTitulos) CancelarPorOrigem(_ context.Context, cancelamento domainreceber.CancelamentoDeOrigem) (domainreceber.ResultadoCancelamento, error) {
	if r.erroDosTitulos != nil {
		return domainreceber.ResultadoCancelamento{}, r.erroDosTitulos
	}
	r.cancelamentos = append(r.cancelamentos, cancelamento)

	resultado := domainreceber.ResultadoCancelamento{Cancelados: []uuid.UUID{}, Marcados: []uuid.UUID{}}
	for _, origemID := range cancelamento.OrigemIDs {
		titulo, ok := r.titulosPorOrigem[origemID]
		if !ok {
			resultado.Ignorados++
			continue
		}
		resultado.Cancelados = append(resultado.Cancelados, titulo)
		delete(r.titulosPorOrigem, origemID)
	}

	return resultado, nil
}
