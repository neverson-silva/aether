# Aether Design System

## 1. Identidade

O Aether Design System é a linguagem visual, comportamental e técnica do Aether, um PaaS para pessoas que constroem, operam e observam infraestrutura digital. Ele não é uma coleção de componentes bonitos, nem um tema aplicado sobre telas existentes. É um sistema de decisões compartilhadas que transforma a complexidade operacional do produto em interfaces previsíveis, densas, legíveis e confiáveis.

O sistema precisa comunicar:

- Controle sem parecer burocrático.
- Precisão sem parecer frio ou inacessível.
- Profundidade técnica sem exigir conhecimento desnecessário.
- Estado operacional com clareza imediata.
- Confiança, continuidade e reversibilidade em ações de alto impacto.

A linguagem atual do produto é o ponto de partida: superfícies escuras, azul perolado, contraste de infraestrutura, Inter para interface, JetBrains Mono para dados técnicos e detalhes de grid, brilho e glass usados com moderação. A evolução deve preservar reconhecimento e reduzir inconsistência. Não devemos redesenhar por preferência pessoal quando a decisão pode ser resolvida por uma regra de sistema.

## 2. Missão e escopo

O sistema deve oferecer uma base React instalável para o front do Aether e para futuros produtos que compartilhem o mesmo modelo mental. A biblioteca é responsável por:

- Fundamentos visuais e tokens de tema.
- Componentes interativos e não interativos.
- Primitivas de composição e layout quando houver repetição comprovada.
- Comportamento acessível, foco, teclado, seleção, abertura e fechamento.
- Estados operacionais como loading, erro, vazio, sucesso, warning e atualização em tempo real.
- Documentação executável, exemplos, contratos de API e decisões de uso.

A biblioteca não é responsável por:

- Regras de negócio específicas de um produto.
- Dados, chamadas HTTP, cache, autenticação ou autorização.
- Decidir a arquitetura de páginas consumidoras.
- Esconder problemas de UX em abstrações genéricas.
- Criar um componente para cada variação visual isolada.

Uma solução pertence ao design system quando é recorrente, tem fronteira clara, possui comportamento consistente e reduz decisões repetidas em mais de um fluxo. Uma solução local é preferível quando representa uma regra única de produto ou quando ainda não há evidência suficiente para abstração.

## 3. Princípios de design

### 3.1 Clareza antes de decoração

Hierarquia, contraste, agrupamento, rótulos e feedback devem funcionar sem gradientes, glow ou glass. Efeitos visuais são uma camada de personalidade, nunca a única forma de comunicar estado ou importância.

### 3.2 Densidade com legibilidade

Um PaaS precisa apresentar tabelas, métricas, logs, eventos e controles em pouco espaço. Densidade não significa reduzir tudo ao menor tamanho possível. Cada redução de espaço ou tipografia deve preservar leitura, foco, alvo de toque e escaneabilidade.

### 3.3 Estado é conteúdo

Loading, erro, vazio, stale, offline, sucesso e atualização são partes da experiência, não exceções de implementação. Componentes e padrões devem tornar estados importantes visíveis e explicar a próxima ação possível.

### 3.4 Segurança para ações destrutivas

Ações de deploy, rollback, delete, rotate, scale e mudança de ambiente devem comunicar consequência, escopo e reversibilidade. O design system fornece padrões para confirmação, pending, sucesso e falha, mas não deve usar modal como solução automática para toda ação perigosa.

### 3.5 Composição sobre configuração infinita

Um componente deve ter uma API pequena e coerente. Quando a quantidade de booleans, slots e combinações inválidas cresce, separar componentes ou adotar composição explícita é preferível a criar uma “API universal”.

### 3.6 Acessibilidade como contrato

Acessibilidade não é uma revisão posterior. Nome acessível, semântica, teclado, foco, contraste, movimento e leitura por tecnologia assistiva fazem parte da definição de pronto de cada componente.

### 3.7 Compatibilidade deliberada

O dark mode existente é uma dependência de produto e não pode regredir. O light mode deve ter equivalência semântica, não necessariamente os mesmos valores numéricos. Um tema muda valores; não muda anatomia, API ou fluxo.

## 4. Modelo técnico

### 4.1 Stack

- React + TypeScript como plataforma principal.
- Vite para desenvolvimento e build de biblioteca.
- Tailwind CSS v4 para composição e geração utilitária.
- Storybook como documentação executável e laboratório de estados.
- `@base-ui/react` como base comportamental real para Button, Toolbar, Dialog, Popover, Menu, Select, Combobox, Tooltip e outras interações complexas.
- `@phosphor-icons/react` como biblioteca oficial e única fonte padrão de ícones.
- React e React DOM como peer dependencies.

### 4.2 Limites de camada

O sistema possui quatro camadas. A dependência deve fluir de cima para baixo, nunca no sentido inverso:

1. **Primitives**: valores brutos de cor, alpha, escala, fontes e medidas. Não são API de componente.
2. **Semantics**: intenção de interface, como canvas, content-muted, surface-overlay, action-primary, focus-ring e status-danger.
3. **Component contracts**: decisões específicas do componente, como button-primary-bg, input-border-invalid e dialog-surface.
4. **Components and patterns**: React, composição, comportamento, markup e stories.

Primitives podem alimentar semantics. Semantics podem alimentar contracts. Components não podem consumir primitives diretamente. A regra evita que um componente escolha `#b0c6ff` ou `rgba(...)` e crie uma decisão impossível de sincronizar entre temas.

### 4.3 Organização de arquivos

Cada componente deve ficar em um diretório próprio e ter um único arquivo de implementação por componente:

```text
src/components/<component>/<component>.tsx
src/components/<component>/<component>.stories.tsx
```

É proibido concentrar componentes diferentes em arquivos genéricos como `foundation.tsx`, `primitives.tsx`, `components.tsx` ou `index.tsx`. Arquivos adicionais só devem existir quando houver uma fronteira real, como hooks, tipos ou estilos compartilhados. Exports públicos ficam em `src/index.ts`. Imports profundos de diretórios internos não são API suportada.

## 5. Teoria de tokens

### 5.1 Token não é variável conveniente

Um token representa uma decisão nomeada, reutilizável e governada. `--color-blue-400` é uma primitive. `--color-action-primary` é um token semântico. `--component-button-primary-bg` é um contrato de componente. O segundo e o terceiro explicam por que o valor existe; o primeiro apenas descreve o que ele é.

### 5.2 Primitives

Primitives contêm a matéria-prima do sistema:

- Escalas neutras para canvas, superfícies e conteúdo.
- Escalas de azul para ações, foco e destaque Aether.
- Escalas de indigo e laranja para papéis secundários e terciários.
- Escalas de vermelho para erro e perigo.
- Alpha para glass, bordas, overlays e elevação.
- Famílias e pesos de fonte.
- Escalas de espaço em múltiplos consistentes de 4px.
- Raios, elevações, durações e curvas de movimento.

Primitives não devem ser escolhidas diretamente por uma tela ou componente. Se uma primitive é usada em três lugares com a mesma intenção, ela precisa ser promovida ou referenciada por um semantic token.

### 5.3 Semantics

Os papéis semânticos devem responder a perguntas de design:

- Qual é o canvas da aplicação?
- Qual superfície está no primeiro, segundo ou terceiro nível de elevação?
- Qual conteúdo é principal, secundário ou desabilitado?
- Qual borda é estrutural, sutil, forte ou de foco?
- Qual cor representa ação primária, ação quieta, seleção ou perigo?
- Qual feedback representa erro, warning, sucesso, informação e estado ao vivo?

A hierarquia de superfície deve ser relacional. `surface-2` não significa “cinza número 2”; significa uma superfície que precisa se destacar de `surface-1` sem competir com um overlay. A mesma relação deve existir no light e no dark, mesmo quando os valores mudarem de claro para escuro.

### 5.4 Tema

O tema é uma tabela de valores para os mesmos papéis. A anatomia e a semântica permanecem estáveis:

- `light`: canvas claro, superfícies claras, conteúdo escuro, bordas visíveis e ações com contraste.
- `dark`: canvas escuro, superfícies elevadas progressivamente, conteúdo claro, bordas de baixa intensidade e primary perolado.

Não criar `ButtonDark`, `CardLight` ou classes específicas de tela. A exceção visual deve ser resolvida com token semântico ou composição local justificada.

### 5.5 Contraste e não dependência de cor

Cor não pode ser o único canal de informação para erro, seleção, estado live ou sucesso. Associar cor a texto, ícone, forma, posição ou padrão. Validar contraste para texto normal, texto grande, controles, focus ring e estados desabilitados sem remover completamente a percepção do controle.

## 6. Tipografia, espaço e movimento

### 6.1 Tipografia

Inter é a fonte de interface. JetBrains Mono é reservada para código, IDs, comandos, logs, timestamps técnicos e valores que se beneficiam de alinhamento monoespaçado. Não usar fonte monoespaçada como decoração.

Os estilos devem expressar função, não apenas tamanho:

- Display: orientação e entrada de área, usado raramente.
- Heading: estrutura de seção e hierarquia.
- Body: leitura e descrição.
- Label: controles, tabelas, filtros e metadados.
- Code: dados técnicos, comandos e valores operacionais.

Evitar criar um novo tamanho para resolver uma hierarquia que pode ser resolvida com peso, espaço, contraste ou agrupamento.

### 6.2 Espaçamento

Usar a escala base de 4px. Espaço interno, distância entre controles e separação de seções devem seguir uma lógica consistente:

- 4px e 8px para relações internas estreitas.
- 16px para agrupamento de controles e conteúdo.
- 24px para blocos relacionados.
- 40px ou mais para mudança de contexto.

Valores arbitrários são aceitáveis apenas quando resolvem uma necessidade de layout documentada, não para ajustar visualmente um caso isolado.

### 6.3 Movimento

Movimento deve comunicar causalidade: entrada, saída, mudança de estado, progresso ou relação espacial. Não animar apenas para tornar a interface “viva”. Toda animação precisa funcionar com `prefers-reduced-motion: reduce`. Estados de loading não podem depender exclusivamente de movimento.

### 6.4 Microinterações como contrato de UX

Microinteração não é ornamento. No Aether, ela é a camada temporal que explica a relação entre intenção, ação, resposta e consequência. Uma interface operacional sem feedback de transição parece quebrada mesmo quando o estado interno está correto. Uma interface com movimento sem causalidade parece instável e reduz a confiança em ações de deploy, rollback, scale, delete e navegação entre ambientes.

Toda interação relevante deve responder a quatro perguntas:

- O que mudou por causa da ação do usuário?
- Em que direção ou região a mudança aconteceu?
- Quanto tempo o usuário precisa para perceber a mudança?
- O que acontece se a ação for interrompida, repetida, revertida ou bloqueada?

Uma contribuição não está pronta se o estado final está correto, mas o caminho até ele é seco, instantâneo ou visualmente ambíguo. O componente deve ter uma resposta de entrada, uma resposta de press ou foco quando aplicável, uma resposta de alteração de estado e uma saída coerente. Essas respostas devem ser discretas em controles densos e mais expressivas em mudanças de contexto.

#### Hierarquia temporal

Usar a escala de motion do sistema como ponto de partida:

- Instantâneo: até 100ms para pressed, mudança de cor, opacidade ou feedback de teclado.
- Rápido: 120ms a 180ms para hover, focus, seleção, expansão curta e feedback de campo.
- Normal: 200ms a 280ms para menus, popovers, tooltips, troca de conteúdo e transições de superfície.
- Ênfase: 300ms a 420ms para drawers, sheets, mudanças de contexto e feedback de sucesso que precise ser percebido.
- Prolongado: acima de 420ms somente para progresso real, entrada de dados complexa ou movimento espacial que o usuário acompanha.

Essas faixas não são valores arbitrários para serem repetidos sem pensar. A duração deve considerar distância, massa percebida, importância da ação e frequência de uso. Um botão usado centenas de vezes por hora não deve ter a mesma duração de um drawer que muda o contexto da tela.

#### Direção e causalidade

- Um drawer lateral entra da lateral em que será ancorado e sai pela mesma direção.
- Um bottom sheet entra de baixo e não deve aparecer como um modal que cresce no centro.
- Um popover nasce próximo ao trigger, preservando a relação espacial.
- Um item inserido em uma lista ocupa sua posição com uma transição curta; não deve deslocar todos os itens sem feedback.
- Um painel redimensionável deve acompanhar o ponteiro durante o drag, sem atraso perceptível.
- Uma troca de seleção deve alterar o indicador no próprio controle, sem mover a página inteira.
- Um toast entra da região em que permanecerá e não atravessa a tela sem motivo.
- Um conteúdo removido deve desaparecer antes de o espaço ser reorganizado quando isso ajudar o usuário a entender a consequência.

Não usar uma animação genérica de `fade-in` para todos os contextos. Opacidade sozinha comunica existência, mas não comunica origem, destino ou relação espacial. Combinar opacity com transform pequeno, cor, borda ou elevação apenas quando cada propriedade tiver função perceptível.

#### Estados de interação

Cada componente interativo deve considerar, quando aplicável:

- Idle: nenhuma transição contínua e nenhum brilho permanente sem propósito.
- Hover: resposta rápida, reversível e que não desloca o layout.
- Focus-visible: foco persistente, contrastante e independente de hover.
- Pressed: resposta imediata, normalmente com cor, sombra, escala mínima ou deslocamento de 1px.
- Dragging: o elemento deve acompanhar o ponteiro, capturar a interação e expor cursor e estado visual.
- Opening: origem, direção e destino devem ser claros.
- Open: o conteúdo deve permanecer estável e não pulsar sem informação nova.
- Closing: preservar o contexto até a saída terminar; não desmontar cedo demais quando isso cortar a animação.
- Loading: manter a identidade e o label da ação, impedindo duplicidade sem apagar contexto.
- Success, warning e error: o feedback deve ser percebido sem depender apenas de cor ou animação.
- Disabled: não animar como se a ação estivesse disponível e não permitir eventos acidentais.

#### Movimento durante interação direta

Interações diretas, como resize, drag, reorder, slider e swipe, não devem ser implementadas como uma série de estados discretos. O valor visual precisa acompanhar o ponteiro ou o dedo continuamente, com limites claros e sem transição que cause atraso. Durante o gesto:

- Usar `pointerdown`, `pointermove` e `pointerup` ou uma primitive headless adequada.
- Capturar o ponteiro para que a interação não quebre ao sair do handle.
- Impedir seleção de texto e scroll concorrente quando isso for necessário.
- Expor alternativa de teclado para qualquer operação que tenha drag.
- Mostrar estado dragging, cursor adequado e foco visível.
- Respeitar `min`, `max`, snap points e valores controlados.
- Emitir mudanças de valor de forma previsível, sem depender de `onMouseMove` apenas.
- Liberar listeners e captura quando a interação termina ou o componente desmonta.

#### Interrupção, reversão e concorrência

Uma microinteração precisa tolerar uma segunda ação antes do término da primeira. Não acumular timers, não deixar duas animações brigando pela mesma propriedade e não impedir Escape, click outside ou nova seleção porque uma saída ainda está em execução. Quando a causa muda, a transição deve partir do estado visual atual ou usar uma estratégia de presença que preserve continuidade.

Componentes assíncronos devem diferenciar a animação de espera da resposta do servidor. Não usar uma animação para sugerir sucesso antes de a operação ter terminado. Em erro, preservar o contexto e orientar a próxima ação. Em sucesso, não remover imediatamente a evidência necessária para confirmar o resultado.

#### Tokens e implementação

Motion deve usar tokens de duração e easing, classes Tailwind semânticas e atributos de estado fornecidos pela primitive. Não espalhar números de duração em JSX sem uma decisão documentada. Não usar `transition-all` por padrão em componentes complexos, porque ele pode animar layout, dimensões e propriedades não intencionais. Preferir propriedades explícitas como `transition-[transform,opacity]`, `transition-colors` e `transition-[width]`.

Base UI deve fornecer o comportamento complexo de presença, focus management, dismiss, swipe e portal quando aplicável. O Aether fornece a linguagem visual através de `data-starting-style`, `data-ending-style`, `data-open`, `data-closed`, `data-swiping` e atributos equivalentes. Nunca remover uma primitive acessível e reimplementar apenas para obter uma animação mais simples.

#### Reduced motion

`prefers-reduced-motion: reduce` não significa remover feedback. Significa remover deslocamento, escala, parallax, shimmer e loops contínuos quando eles não são essenciais, mantendo mudança de estado por cor, borda, conteúdo, foco e opacity curta. Uma pessoa que reduz movimento ainda precisa saber que um drawer abriu, que um item foi selecionado e que uma operação terminou.

Stories devem demonstrar o comportamento normal e reduced motion. Toda animação contínua deve ter uma razão de produto, uma alternativa estática e um limite de performance. O componente não pode depender de `animationend` para funcionar corretamente.

#### Definition of Done de motion

Antes de promover um componente, verificar:

1. Entrada e saída têm causalidade e direção.
2. Hover, focus-visible, pressed e disabled não causam layout shift inesperado.
3. Drag e resize acompanham o ponteiro e possuem teclado equivalente.
4. Fechamento não corta conteúdo nem remove foco de maneira abrupta.
5. Interações repetidas e interrompidas não deixam estado preso.
6. Duração, easing e propriedades animadas são intencionais.
7. Reduced motion mantém feedback funcional.
8. Light e dark mantêm contraste durante a transição.
9. A story mostra o estado inicial, a interação e o resultado.
10. A animação não mascara erro, loading, permissão ou confirmação.

## 7. Componentes React

### 7.1 API prop-driven

O padrão preferencial do Aether é uma API orientada a props, não uma Composition API. Componentes simples e médios devem resolver seu caso de uso recebendo as decisões necessárias diretamente, com tipos claros e defaults previsíveis.

Exemplos preferenciais:

- `Button` recebe `variant`, `size`, `loading`, `disabled`, `icon`, `iconPosition` e `as` quando aplicável.
- `Input` recebe `label`, `description`, `error`, `leadingIcon`, `trailingIcon`, `clearable`, `loading` e `size`.
- `Badge` recebe `tone`, `size`, `icon`, `dot` e `onRemove`.
- `Avatar` recebe `src`, `fallback`, `status`, `size` e `alt`.
- `EmptyState` recebe `icon`, `title`, `description`, `action` e `secondaryAction`.

Phosphor Icons é a biblioteca padrão do Aether. Não adicionar Material Symbols, lucide, react-icons, SVGs avulsos ou outra família para resolver um caso local sem uma decisão explícita de arquitetura. O pacote oficial atual é `@phosphor-icons/react`; não usar o pacote legado `phosphor-react`.

Ícones devem ser aceitos como props tipadas, usando os tipos e componentes de `@phosphor-icons/react`, sem exigir que o consumidor monte wrappers internos para posicioná-los. A API deve definir posição, tamanho, alinhamento, peso e comportamento decorativo do ícone. Ícones decorativos devem usar `aria-hidden`; ícones que são a única representação de uma ação exigem nome acessível, normalmente via `aria-label` ou tooltip.

Usar os weights do Phosphor de forma consistente. `regular` é o default de interface; `bold` e `fill` devem comunicar peso, seleção ou estado e não ser escolhidos arbitrariamente. `duotone` e `thin` são exceções de contexto e precisam respeitar contraste e legibilidade. O tamanho do ícone deve acompanhar o tamanho do controle e nunca ser usado apenas para preencher espaço.

Quando um componente receber `icon`, `leadingIcon` ou `trailingIcon`, o contrato deve aceitar o componente ou elemento Phosphor e controlar o layout internamente. Não obrigar o consumidor a usar `Button.Icon`, `Input.Icon` ou wrappers equivalentes em casos simples.

Não transformar um componente simples em uma árvore de subcomponentes apenas para fornecer ícone, label, descrição ou ação. O consumidor não deve precisar conhecer a anatomia interna para realizar uma operação comum.

Composition API é uma exceção justificada para estruturas complexas e repetíveis, como Dialog, Menu, Tabs, Table avançada, Sidebar e layouts com regiões independentes. Mesmo nesses casos, oferecer presets e atalhos prop-driven para os casos mais comuns.

### 7.2 API

Cada componente deve:

- Ter props tipadas e nomes orientados ao domínio do componente.
- Expor variantes finitas, documentadas e mutuamente compreensíveis.
- Permitir `className` apenas como escape hatch controlado, sem exigir sobrescrita de estilos internos.
- Preservar props nativas quando o elemento base for nativo.
- Encaminhar `ref` quando consumidores precisarem focar, medir ou integrar com uma primitive.
- Não esconder conteúdo, estado ou regra de negócio dentro da apresentação.
- Evitar booleanos que criem combinações inválidas.

Quando a composição for realmente necessária, preferir subcomponentes explícitos, como `Dialog.Root`, `Dialog.Trigger` e `Dialog.Content`, mas não impor essa estrutura a componentes que podem ter uma API prop-driven simples. Antes de criar slots ou subcomponentes, provar que props criariam combinações inválidas, conteúdo arbitrário ou regiões independentes.

### 7.3 Variants

Muitos componentes do Aether terão variants para representar intenções e estados previsíveis, como `variant`, `size`, `tone`, `orientation` e `density`. Toda definição de variants deve usar `tailwind-variants` (`tv`), mantendo a matriz de estilos em um único contrato tipado.

Regras para variants:

- Definir `base`, `variants`, `defaultVariants` e, quando necessário, `compoundVariants` dentro da configuração do componente.
- Manter cada eixo de variant com uma responsabilidade única. Não usar `size` para resolver cor ou `tone` para resolver espaçamento.
- Preferir nomes de intenção (`primary`, `danger`, `quiet`, `selected`) a nomes de cor ou implementação.
- Usar `compoundVariants` somente quando a combinação representar uma regra real do componente.
- Não espalhar ternários de classes, mapas manuais de classes ou concatenação condicional paralela ao `tv`.
- Variants devem ser tipadas e fazer parte da API documentada do componente.
- Não criar variants para exceções de uma única tela; usar composição ou `className` local quando a necessidade não for sistêmica.
- A classe externa deve ser aplicada pelo resultado do `tv`, preservando a precedência e a composição previstas pela biblioteca.
- Estados comportamentais como loading e disabled só devem virar variant quando também possuírem uma expressão visual própria; comportamento e acessibilidade continuam sendo tratados pelo componente.

Antes de adicionar um novo eixo de variant, verificar se ele representa uma decisão repetível, se suas combinações são válidas e se todos os estados resultantes podem ser documentados no Storybook.

### 7.4 Semântica HTML

Usar elemento nativo sempre que ele já representar o comportamento: `button`, `a`, `input`, `label`, `select`, `table`, `nav`, `main`, `header` e `section`. `div` com `role="button"` só é aceitável quando a composição torna o elemento nativo inviável e todo o contrato de teclado é implementado.

Não usar `aria-label` para compensar texto visual ausente quando um label visível é necessário. Não adicionar ARIA redundante. O nome acessível deve refletir a ação ou o conteúdo real.

### 7.5 Primitivas headless

Usar Base UI quando o comportamento envolver foco, portal, posicionamento, roving tabindex, typeahead, dismiss, nested interactions ou gerenciamento complexo de teclado. O Aether deve estilizar a primitive e manter a API visual do design system. O uso deve aparecer no código do componente, não apenas na documentação.

Button e IconButton usam `@base-ui/react/button`; ButtonGroup usa `@base-ui/react/toolbar`. Componentes futuros de Dialog, Popover, Menu, Select, Combobox e Tooltip devem usar suas primitives Base UI correspondentes. Componentes simples que usam HTML nativo não precisam de Base UI artificialmente.

Não importar uma primitive apenas para um comportamento trivial que o HTML nativo resolve. Não duplicar internamente lógica já fornecida por Base UI. Se uma limitação da primitive exigir workaround, documentar a decisão na story ou no registro de arquitetura.

### 7.5.1 Busca assíncrona

`SelectSearch` e `Combobox` com opções locais são adequados somente quando o conjunto completo é pequeno, está disponível e pode ser filtrado no cliente. Não usar essa API para recursos, serviços, projetos, ambientes, logs ou usuários em catálogos potencialmente grandes.

Para busca remota, usar `AsyncSearchInput` ou um componente de domínio baseado nele. A API deve receber o termo e buscar resultados incrementais através de `loadOptions(query)`, sem exigir que o consumidor pré-carregue todos os registros.

O contrato de busca assíncrona deve:

- Aplicar debounce configurável.
- Ignorar respostas antigas quando uma consulta mais nova já foi emitida.
- Expor loading no campo e na lista.
- Expor erro recuperável e permitir nova tentativa digitando novamente.
- Ter `minQueryLength` para evitar chamadas vazias e consultas excessivas.
- Diferenciar “digite mais caracteres”, “carregando”, “sem resultados” e “erro”.
- Preservar teclado, highlight, seleção e foco através da primitive Base UI.
- Permitir descrições e metadados nos resultados sem depender apenas de cor.
- Não esconder a latência nem simular resultados antes da resposta real.
- Cancelar timers e limpar efeitos quando o componente desmontar.

Não fazer fetch diretamente em `render`, não usar uma lista completa como fallback silencioso e não manter resultados de uma query antiga quando eles podem ser confundidos com a query atual. Stories devem simular latência, erro, nenhum resultado, query curta, resultado selecionado e mudança rápida de consulta.

### 7.5.2 Feedback e resiliência

Toast é confirmação curta de uma ação e deve desaparecer sem exigir leitura posterior. Notification é informação persistente, pode acumular eventos e precisa de consulta, agrupamento ou marcação de lida. Banner permanece associado a uma condição ampla da aplicação. Inline Error permanece junto da área que falhou. Não usar um contrato no lugar do outro.

Todo feedback de rede deve distinguir loading, erro, offline, reconnecting, stale, queued mutation, sync success e sync conflict quando esses estados forem possíveis. Loading não deve bloquear a aplicação inteira quando apenas uma seção está carregando. Erros devem preservar contexto, oferecer retry quando a operação for repetível e expor request ID ou report ID sem vazar dados sensíveis.

Toast e Notification Stack devem respeitar limite de simultaneidade, prioridade, timeout, persistência, pause on hover e reduced motion. Não usar animação para anunciar sucesso antes da confirmação real. Estados de offline e reconnecting precisam de texto e ícone; um ponto colorido isolado não é suficiente.

### 7.5.3 Fluxos de produto

Wizard, Questionnaire, Form Builder, Resource Picker, Environment Switcher, Deployment Composer, Variable Editor, Command Runner, Diff Viewer, Approval Flow, Bulk Action Bar, Saved View, Activity Feed e Changelog são padrões de apresentação e interação. Eles não devem buscar dados, decidir permissões, executar deploy, persistir draft ou interpretar regras de negócio.

Esses fluxos devem receber dados, schema, estados e callbacks por props. Estados de rede, validação, autorização, dirty state, autosave, aprovação e rollback devem ser explícitos na API. Um componente não pode transformar uma falha remota em estado de sucesso, esconder uma ação sem informar que ela foi bloqueada ou executar uma operação destrutiva sem que o consumidor forneça confirmação e consequência.

Fluxos longos devem preservar contexto durante navegação entre etapas, permitir back sem perder dados, comunicar progresso e tratar abandono. Autosave deve ser representado como estado observável (`saving`, `saved`, `error`), nunca como um efeito silencioso. Preview, review, diff e summary devem permitir inspeção antes de uma ação irreversível.

### 7.5.4 Interação avançada

Componentes de exploração devem manter uma alternativa não gestual. Carousel precisa de controles, dots e teclado além de autoplay ou swipe. Drag and drop precisa de operação por teclado, preview, cancelamento, alvo inválido e feedback explícito. Resize e reorder devem acompanhar o ponteiro sem atraso e nunca depender apenas de hover.

Virtualização deve preservar scroll anchoring, foco e leitura assistiva. Quando linhas dinâmicas impedirem cálculo seguro, oferecer fallback não virtualizado ou estratégia de medição documentada. Não desmontar o item focado sem transferir foco de forma previsível.

Direction Provider é parte da infraestrutura: não codificar `left` e `right` como semântica quando a intenção for `start` e `end`, não inverter ícones arbitrariamente em RTL e não assumir que o primeiro item visual é sempre o primeiro item lógico. Realtime deve diferenciar conectado, pausado, desconectado, unread, backfill e conflito; pulse ou cor sozinhos não comunicam esses estados.

### 7.6 Estados

O contrato de estado deve distinguir:

- `disabled`: ação indisponível por regra ou permissão.
- `loading`: ação em andamento; preservar contexto e impedir submissão duplicada.
- `pending`: estado aguardando confirmação, rede ou processamento externo.
- `error`: operação ou campo inválido, com mensagem acionável.
- `empty`: não há conteúdo; explicar por quê e qual próximo passo existe.
- `selected`: escolha persistente ou estado de filtro.
- `active`: interação atual, não confundir com selected.
- `readonly`: conteúdo visível, mas não editável.
- `stale` ou `offline`: dados disponíveis, mas potencialmente desatualizados.

Não usar opacidade como única diferença entre estados. Não representar loading trocando o texto por uma palavra genérica se isso destruir o contexto da ação.

## 8. Storybook como especificação

Storybook é parte do produto da biblioteca. Uma story não é apenas uma imagem de exemplo; é um caso verificável de API, comportamento e acessibilidade.

Todo componente deve ter `*.stories.tsx`, `tags: ['autodocs']` e documentação suficiente para responder:

- O que o componente resolve?
- Quando deve ser usado e quando não deve?
- Qual é sua anatomia?
- Quais variantes existem e qual é a intenção de cada uma?
- Quais estados são suportados?
- Como funciona em light e dark?
- Como funciona com conteúdo curto, longo, ausente e dinâmico?
- Como funciona em viewport estreita?
- Como é operado por teclado e lido por tecnologia assistiva?

### Matriz mínima de stories

Cada componente deve cobrir, quando aplicável:

1. Default e caso de uso principal.
2. Todas as variantes públicas.
3. Todos os tamanhos e densidades.
4. Hover, focus-visible, active e pressed.
5. Disabled, readonly e loading.
6. Error, warning, success e live.
7. Empty, selected e conteúdo longo.
8. Light e dark.
9. Layout mobile e overflow controlado.
10. Composição com label, descrição, ícone e mensagem de erro.

Estados não aplicáveis devem ser explicados. Não criar stories artificiais apenas para preencher uma lista.

Stories devem usar controles para props relevantes, dados realistas, ações observáveis e addon de acessibilidade. Não esconder estados importantes em uma única story com muitas combinações impossíveis de descobrir.

### Taxonomia canônica do Storybook

O Storybook do Aether possui uma árvore pública única. A intenção de uso não deve criar uma nova seção para um componente que já pertence a outra categoria. Se o mesmo componente serve a deploy, billing e observabilidade, essas diferenças devem ser stories dentro do mesmo componente, não cópias em seções de produto.

Os únicos grupos de primeiro nível permitidos são:

- `Foundations`: tokens, tema, tipografia, layout, direção e primitives sem intenção de produto.
- `Components`: componentes básicos e reutilizáveis, como Button, Badge, Avatar, Card, Code Block, Marker e Bubble.
- `Forms`: campos, seleção, entrada, filtros e controles de formulário.
- `Navigation`: menus, command palette, spotlight, navegação contextual e user menu.
- `Overlay`: dialog, alert dialog, drawer, sheet, popover, tooltip e hover card.
- `Data`: table, chart, metric, timeline, logs, diff, resource tree, status operacional e virtualização.
- `Feedback`: alertas, banners, mensagens, toasts, loading, erros, offline e feedback persistente.
- `Patterns`: fluxos compostos de produto, como wizard, deployment composer, approval flow, activity feed e dashboards.

Regras de nomenclatura:

- O `title` deve seguir exatamente `Grupo/Nome do componente`, com o nome canônico da API pública.
- Não usar `Flows`, `Advanced`, `Infrastructure`, `Application Structure` ou `Forms and Input` como grupos de primeiro nível.
- Não criar uma segunda story canônica para o mesmo componente em outro grupo. Variantes de intenção ficam sob o mesmo título.
- Stories de showcase agrupadas, como `Foundations/Primitives` e `Forms/Overview`, são permitidas apenas como índice visual; cada componente interativo continua obrigado a ter sua story própria.
- O caminho físico do arquivo não define a categoria. A categoria é definida pelo contrato do componente e pelo `title` explícito.
- Renomear um `title` altera o ID de navegação; mudanças desse tipo devem ser feitas em conjunto e validadas pelo `index.json` gerado.

## 9. Acessibilidade e qualidade de interação

Definition of Done de qualquer componente interativo:

- O controle tem nome acessível.
- A ordem de tabulação é lógica.
- O foco é visível em light e dark.
- Enter, Space, Escape, setas e typeahead funcionam quando o padrão exigir.
- O estado atual é anunciado ou exposto semanticamente.
- Mensagens de erro são associadas ao campo ou região correta.
- Não existe dependência exclusiva de cor, hover ou movimento.
- O componente tolera zoom, conteúdo longo e labels traduzidos.
- Alvos de toque têm tamanho adequado ao contexto.
- Contraste e reduced motion foram verificados.

## 10. Governança

### 11.1 Mudanças pequenas

Uma mudança pequena pode alterar implementação sem mudar o contrato visual ou público. Ainda assim, deve atualizar story e testes quando houver mudança de comportamento.

### 11.2 Mudanças de token

Alteração de token deve informar:

- Qual papel semântico mudou.
- Quais temas foram afetados.
- Qual contraste ou hierarquia mudou.
- Quais componentes e consumidores podem mudar visualmente.
- Se é correção, evolução ou breaking change.

Nunca substituir um token semântico por outro apenas porque os valores atuais são iguais. Igualdade de valor não significa igualdade de intenção.

### 11.3 API e versionamento

- Adição compatível de prop ou componente: minor.
- Correção sem alteração pública: patch.
- Remoção, renomeação, mudança de semântica ou alteração incompatível de markup: major.
- Depreciações precisam de alternativa, período de migração e registro no changelog.

O pacote deve expor apenas o que está em `src/index.ts`. Um arquivo interno não é público só porque pode ser importado pelo bundler.

### 11.4 Critério de promoção

Um padrão local só deve virar componente compartilhado quando houver repetição observável, API estável, estados conhecidos e pelo menos dois consumidores ou fluxos previstos. Abstrair cedo demais transforma hipóteses em dívida de API.

## 11. Anti-padrões

- Copiar tokens de uma tela para outra sem definir intenção.
- Usar hex, rgba ou gradientes diretamente em JSX.
- Criar variantes para cada combinação de cor, tamanho e contexto.
- Usar `!important` para compensar uma API mal definida.
- Reimplementar dialog, menu, tooltip ou combobox acessível sem necessidade.
- Usar modal para comunicar toda ação irreversível.
- Fazer um componente depender de estado global de negócio.
- Ocultar conteúdo com `display: none` quando o leitor precisa receber a informação.
- Criar um estado “disabled” que continua clicável.
- Fazer loading deslocar layout ou perder o label da ação.
- Usar animação, glow ou cor como única indicação de estado.
- Adicionar comentários ao código. A intenção deve estar expressa em nomes, tipos, estrutura e documentação do sistema.

## 12. Processo de implementação

1. Descrever problema, consumidor, frequência e limite do componente.
2. Definir anatomia, semântica HTML, estados, teclado, foco e responsividade.
3. Mapear cada necessidade aos tokens semânticos existentes.
4. Definir API mínima e combinações válidas.
5. Escolher HTML nativo ou primitive headless.
6. Implementar comportamento antes de efeitos visuais.
7. Implementar composição Tailwind usando classes registradas no `@theme`, sem classes arbitrárias baseadas em variáveis CSS dentro dos componentes.
8. Criar stories para a matriz de estados.
9. Validar acessibilidade, light, dark, conteúdo extremo e reduced motion.
10. Exportar somente a API aprovada.
11. Registrar impacto, migração e versão.

## 13. Resultado esperado

Uma contribuição está pronta quando outro time consegue instalá-la, entender sua intenção, compô-la sem conhecer detalhes internos, operar todos os estados com teclado, visualizar seus limites no Storybook e confiar que ela mantém a mesma semântica nos temas claro e escuro.
