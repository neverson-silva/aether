---
name: export-frontend
description: Especialista em transformar designs, screenshots, protótipos e especificações em frontend funcional. Use quando a tarefa pedir exportar ou implementar uma interface em HTML/CSS, React, Vue, Next.js ou stack existente, preservando comportamento, responsividade, acessibilidade e integração com o código do projeto.
---

# Export Frontend

## Workflow

1. Inspecione o repositório antes de editar: framework, scripts, rotas, componentes, tokens, assets, convenções e estado atual do git. Preserve padrões existentes; não introduza uma stack nova sem necessidade.
2. Converta a referência em requisitos verificáveis: regiões da tela, estados, interações, breakpoints, conteúdo, dados e comportamento de erro/loading/empty. Se algo estiver ambíguo, registre a hipótese e escolha a alternativa mais reversível.
3. Implemente por camadas: estrutura semântica, layout responsivo, tipografia e tokens, estados/interações e integração com dados. Prefira componentes reutilizáveis e CSS coerente com o projeto.
4. Trate o design como referência, não como imagem a ser falsificada: use HTML real, texto selecionável, foco de teclado, controles nativos quando adequados e assets existentes. Não esconda problemas com screenshots ou valores hardcoded desnecessários.
5. Valide em viewport mobile e desktop. Confira overflow, hierarquia visual, contraste, foco, navegação por teclado, labels, reduced motion e estados sem dados. Execute lint, typecheck, testes e build disponíveis.

## Regras de decisão

- Se já existir um design system, use seus tokens e componentes antes de criar variantes.
- Se a referência for apenas screenshot, não invente lógica de negócio sem declarar a hipótese.
- Se o projeto usa dados remotos, preserve loading, erro, retry e cache conforme o padrão existente.
- Se uma dependência nova for inevitável, explique custo, alternativa sem dependência e impacto no bundle.
- Corrija primeiro problemas funcionais e de acessibilidade; depois ajuste fidelidade visual.

## Entrega

Relate arquivos alterados, decisões/hipóteses, estados implementados, comandos de validação e eventuais limitações visuais. Não declare fidelidade ou testes quando não tiver verificado.
