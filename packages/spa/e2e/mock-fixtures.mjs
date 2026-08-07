// Deterministic API fixtures for the render harness (e2e/shot.mjs). No backend
// or GitLab needed: Playwright intercepts /api/** and serves these. The scenario
// is built to exercise the attention paths — a repo with an awaiting-approval
// review, a failed review, and a paused release — plus a full webhook config.
const NOW = '2026-08-07T10:00:00Z'

const accounts = [{ id: 'acc1', name: 'work', baseUrl: 'https://gitlab.com', createdAt: NOW }]

const providers = [
  {
    id: 'prov1',
    name: 'groq',
    kind: 'openai-compat',
    baseUrl: '',
    model: 'llama-3.3-70b-versatile',
    isDefault: true,
    temperature: null,
    models: ['llama-3.3-70b-versatile', 'moonshot-v1-32k'],
    createdAt: NOW,
  },
]

const repos = [
  {
    id: 'repoA',
    name: 'web-app',
    url: 'https://gitlab.com/acme/web-app',
    accountId: 'acc1',
    providerId: 'prov1',
    model: '',
    webhookEnabled: true,
    webhookRequireConfirmation: false,
    webhookSecret: 'whsec_9f3c1a7b2e4d8065',
    webhookPath: '/api/webhooks/gitlab/repoA',
  },
  {
    id: 'repoB',
    name: 'api-service',
    url: 'https://gitlab.com/acme/api-service',
    accountId: 'acc1',
    providerId: 'prov1',
    model: '',
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '/api/webhooks/gitlab/repoB',
  },
]

const review = (o) => ({
  id: o.id,
  repoId: o.repoId,
  mrIid: o.mrIid,
  contextMode: o.contextMode ?? 'fast',
  model: 'llama-3.3-70b-versatile',
  sourceBranch: 'feat/x',
  targetBranch: 'development',
  status: o.status,
  phase: '',
  archived: false,
  summaryPublished: false,
  summary: o.summary ?? '',
  recommendation: o.recommendation ?? 'comment',
  score: o.score ?? 0,
  error: o.error,
  inputTokens: o.inputTokens ?? 0,
  outputTokens: o.outputTokens ?? 0,
  findings: o.findings ?? [],
  reasonings: [],
  createdAt: o.createdAt ?? NOW,
  updatedAt: o.createdAt ?? NOW,
})

const reviewsRepoA = [
  review({ id: 'rev1', repoId: 'repoA', mrIid: 128, status: 'awaiting_approval', createdAt: '2026-08-07T09:40:00Z' }),
  review({ id: 'rev2', repoId: 'repoA', mrIid: 126, status: 'error', error: 'the model returned an empty response', inputTokens: 1200, createdAt: '2026-08-07T09:20:00Z' }),
  review({
    id: 'rev3',
    repoId: 'repoA',
    mrIid: 120,
    status: 'done',
    recommendation: 'request_changes',
    score: 64,
    summary: 'Two blocking risks and a handful of readability nits. See the findings.',
    inputTokens: 3200,
    outputTokens: 1500,
    findings: [
      { index: 0, dimension: 'risk', severity: 'high', file: 'internal/app/routines/service.go', line: 212, issue: 'Unvalidated tag input flows into the shell command.', why: 'Command injection risk.', fix: 'Validate against the semver regex first.', blocking: true, published: false },
      { index: 1, dimension: 'risk', severity: 'medium', file: 'internal/http/handlers.go', line: 88, issue: 'Error body leaks the internal path.', why: 'Information disclosure.', fix: 'Return a generic message.', blocking: false, published: true },
      { index: 2, dimension: 'readability', severity: 'low', file: 'src/pages/flow.vue', line: 140, issue: 'The withBusy helper name is vague.', why: 'Intent is unclear at call sites.', fix: 'Rename to runReviewAction.', blocking: false, published: false },
      { index: 3, dimension: 'reliability', severity: 'medium', file: 'internal/app/reviews/service.go', line: 514, issue: 'No timeout on the model call.', why: 'A hung provider blocks the worker.', fix: 'Wrap in context.WithTimeout.', blocking: false, published: false },
      { index: 4, dimension: 'resilience', severity: 'low', file: 'internal/adapters/gitlab/client.go', line: 171, issue: 'ListTags has no retry on 5xx.', why: 'Transient GitLab errors fail the run.', fix: 'Add a bounded backoff.', blocking: false, published: false },
      { index: 5, dimension: 'readability', severity: 'low', file: 'src/pages/reviews/index.vue', line: 140, issue: 'Deeply nested computed.', why: 'Hard to follow.', fix: 'Extract a helper.', blocking: false, published: false },
    ],
    createdAt: '2026-08-06T18:00:00Z',
  }),
  // An earlier failed attempt for MR !120 → the list groups it under rev3.
  review({ id: 'rev0', repoId: 'repoA', mrIid: 120, status: 'error', error: 'timeout cloning repo', createdAt: '2026-08-06T17:00:00Z' }),
]

const mrsRepoA = [
  { iid: 128, title: 'Add OpenRouter provider with live model browser', state: 'opened', sourceBranch: 'feat/openrouter', targetBranch: 'development', webUrl: '#', author: 'yahir' },
  { iid: 130, title: 'Fix pagination on tag listing', state: 'opened', sourceBranch: 'fix/tags-pagination', targetBranch: 'development', webUrl: '#', author: 'yahir' },
]

const routineRuns = [
  {
    id: 'run1',
    kind: 'release',
    repoId: 'repoA',
    mrIid: 0,
    status: 'awaiting_confirmation',
    steps: [
      { name: 'compute_tag', status: 'done', detail: 'Next version v1.0.0', updatedAt: NOW },
      { name: 'create_mr', status: 'done', detail: '', updatedAt: NOW },
      { name: 'confirm', status: 'running', detail: '', updatedAt: NOW },
      { name: 'merge', status: 'pending', detail: '', updatedAt: NOW },
      { name: 'tag', status: 'pending', detail: '', updatedAt: NOW },
    ],
    state: { nextTag: 'v1.0.0', lastTag: 'v0.86.1', featCount: 3, fixCount: 2, mrIid: 131, mrTitle: 'Release v1.0.0' },
    lastError: '',
    createdAt: NOW,
    updatedAt: NOW,
    archived: false,
    repoName: 'web-app',
    flow: 'main',
    sourceBranch: 'development',
    targetBranch: 'main',
  },
]

const preflight = {
  tokenScopes: ['api', 'read_repository'],
  scopesKnown: true,
  checks: [
    { capability: 'list_mrs', label: 'List merge requests', status: 'ok', detail: '' },
    { capability: 'post_discussion', label: 'Post inline discussions', status: 'ok', detail: '' },
    { capability: 'merge_mr', label: 'Merge merge requests', status: 'ok', detail: '' },
    { capability: 'push_tag', label: 'Push tags', status: 'unknown', detail: 'Access level could not be read.' },
  ],
}

// Return { status, json } for a matched endpoint, or null to fall through.
export function fixtureFor(method, pathname, search) {
  const p = pathname
  const map = {
    'GET /auth/status': { authEnabled: false, authenticated: false },
    'GET /accounts': accounts,
    'GET /providers': providers,
    'GET /repos': repos,
    'GET /telegram': [],
    'GET /profiles': [],
    'GET /notifications/events': ['review.finished', 'release.finished'],
    'GET /notifications/rules': [],
    'GET /routines': routineRuns.filter((r) => (search.includes('archived=1') ? r.archived : !r.archived)),
  }
  const key = `${method} ${p}`
  if (key in map) return { status: 200, json: map[key] }

  if (method === 'GET' && p === '/repos/repoA/merge-requests') return { status: 200, json: mrsRepoA }
  if (method === 'GET' && p === '/repos/repoB/merge-requests') return { status: 200, json: [] }
  if (method === 'GET' && /^\/repos\/repoA\/reviews$/.test(p))
    return { status: 200, json: search.includes('archived=1') ? [] : reviewsRepoA }
  if (method === 'GET' && /^\/repos\/repoB\/reviews$/.test(p)) return { status: 200, json: [] }
  if (method === 'GET' && /^\/repos\/[^/]+\/routines$/.test(p))
    return { status: 200, json: search.includes('archived=1') ? [] : routineRuns }
  // repoA is missing 'development' (has 'develop') → exercises the branch alert.
  if (method === 'GET' && /^\/repos\/[^/]+\/branches$/.test(p))
    return { status: 200, json: ['main', 'develop', 'feature/openrouter', 'fix/tags-pagination'] }
  if (method === 'GET' && /^\/repos\/[^/]+\/routines\/preview-tag$/.test(p))
    return { status: 200, json: { nextTag: 'v1.0.0', lastTag: 'v0.86.1', featCount: 3, fixCount: 2 } }
  if (method === 'GET' && /^\/repos\/[^/]+\/preflight$/.test(p)) return { status: 200, json: preflight }
  if (method === 'GET' && /^\/reviews\/[^/]+\/humanizations$/.test(p))
    return { status: 200, json: { summary: [], findings: {} } }
  if (method === 'GET' && /^\/reviews\/(.+)$/.test(p)) {
    const id = p.split('/')[2]
    return { status: 200, json: reviewsRepoA.find((r) => r.id === id) ?? reviewsRepoA[2] }
  }
  if (method === 'GET' && /^\/routines\/(.+)$/.test(p)) {
    const id = p.split('/')[2]
    return { status: 200, json: routineRuns.find((r) => r.id === id) ?? routineRuns[0] }
  }
  return null
}
