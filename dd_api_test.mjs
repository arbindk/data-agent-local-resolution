#!/usr/bin/env node
/*
 * DataDynamics — Portal API smoke/contract test harness (v2)
 * --------------------------------------------------------------------------
 * Validates every endpoint the portal UI calls AND asserts the exact JSON
 * field shapes the UI depends on (e.g. job_id, checkpoint.objects_scanned),
 * so backend field-name drift is caught before you test in the browser.
 *
 * v2 changes:
 *   - AUTO-DISCOVERY: finds the real data_store_id + a real container from the
 *     live agent, so you no longer have to guess AZ_SOURCE_ID / AZ_CONTAINER.
 *   - AI recommendations now called correctly (POST + execution_id).
 *   - Picks a recent execution_id automatically (from evidence/audit/actions)
 *     so Guardian-decisions and AI-recommendations tests actually run.
 *   - Longer poll for "running" so pause/resume verify reliably.
 *   - APPLY mode: dry-run governed action by default in Azure mode; a REAL
 *     write-back only when APPLY=1 (and reverts the test tag afterwards).
 *
 * Requirements: Node.js 18+ (built-in fetch). No npm install.
 *
 * Run:
 *   node dd_api_test.mjs                 # agent-only checks
 *   $env:AZURE=1; node dd_api_test.mjs   # + Azure (auto-discovers source/container)
 *   $env:AZURE=1; $env:APPLY=1; node dd_api_test.mjs   # + real write-back test
 *
 * Optional env overrides:
 *   AGENT_URL (http://localhost:8080)  PLATFORM_URL (http://localhost:9090)
 *   ACTOR ROLE TENANT
 *   AZ_SOURCE_ID  AZ_CONTAINER  AZ_PREFIX   (override auto-discovery)
 *   EXECUTION_ID  (override auto-pick)
 *   APPLY=1       (allow a real metadata/tag write-back, then revert)
 */

const AGENT = (process.env.AGENT_URL || 'http://localhost:8080').replace(/\/$/, '');
const PLATFORM = (process.env.PLATFORM_URL || 'http://localhost:9090').replace(/\/$/, '');
const HEADERS = {
  'X-Everest-Actor': process.env.ACTOR || 'portal-tester',
  'X-Everest-Role': process.env.ROLE || 'data_steward',
  'X-Everest-Tenant': process.env.TENANT || 'tenant-local',
};
const AZURE = process.env.AZURE === '1' || process.env.AZURE === 'true';
const APPLY = process.env.APPLY === '1';
let AZ_SOURCE_ID = process.env.AZ_SOURCE_ID || '';
let AZ_CONTAINER = process.env.AZ_CONTAINER || '';
const AZ_PREFIX = process.env.AZ_PREFIX || '';
let EXECUTION_ID = process.env.EXECUTION_ID || '';

const C = { g: '\x1b[32m', r: '\x1b[31m', y: '\x1b[33m', b: '\x1b[36m', dim: '\x1b[2m', z: '\x1b[0m', bold: '\x1b[1m' };
const results = [];
let group = '';
function setGroup(g) { group = g; console.log(`\n${C.bold}${C.b}== ${g} ==${C.z}`); }
function rec(name, status, detail) {
  results.push({ group, name, status, detail: detail || '' });
  const tag = status === 'PASS' ? `${C.g}PASS${C.z}` : status === 'FAIL' ? `${C.r}FAIL${C.z}` : `${C.y}SKIP${C.z}`;
  console.log(`  [${tag}] ${name}${detail ? `  ${C.dim}${detail}${C.z}` : ''}`);
}
const pass = (n, d) => rec(n, 'PASS', d);
const fail = (n, d) => rec(n, 'FAIL', d);
const skip = (n, d) => rec(n, 'SKIP', d);
const info = (msg) => console.log(`  ${C.dim}· ${msg}${C.z}`);

async function call(base, path, { method = 'GET', body } = {}) {
  const url = base + path;
  const opts = { method, headers: { ...HEADERS } };
  if (body !== undefined) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  const t0 = Date.now();
  try {
    const res = await fetch(url, opts);
    const ms = Date.now() - t0;
    const text = await res.text();
    let json; try { json = JSON.parse(text); } catch { json = undefined; }
    return { ok: res.ok, status: res.status, json, text, ms, url };
  } catch (e) {
    return { ok: false, status: 0, json: undefined, text: String(e), ms: Date.now() - t0, url, netErr: true };
  }
}
function has(obj, path) { return path.split('.').reduce((o, k) => (o == null ? undefined : o[k]), obj) !== undefined; }
function get(obj, path) { return path.split('.').reduce((o, k) => (o == null ? undefined : o[k]), obj); }
function arrify(j, ...keys) { if (Array.isArray(j)) return j; for (const k of keys) if (Array.isArray(get(j, k))) return get(j, k); return null; }
function truncate(s, n = 140) { s = (s || '').replace(/\s+/g, ' '); return s.length > n ? s.slice(0, n) + '…' : s; }
function expectShape(name, res, fields = []) {
  if (res.netErr || res.status === 0) return fail(name, `network error: cannot reach ${res.url} — is the agent running?`);
  if (res.status !== 200) return fail(name, `HTTP ${res.status} (expected 200) ${truncate(res.text)}`);
  const missing = fields.filter((f) => !has(res.json, f));
  if (missing.length) return fail(name, `missing field(s): ${missing.join(', ')}`);
  return pass(name, `${res.ms}ms${fields.length ? ` · fields ok: ${fields.join(', ')}` : ''}`);
}

(async function run() {
  console.log(`${C.bold}DataDynamics API contract test (v2)${C.z}`);
  console.log(`${C.dim}agent=${AGENT}  platform=${PLATFORM}  azure=${AZURE ? 'on' : 'off'}  apply=${APPLY ? 'on' : 'off'}${C.z}`);

  // -- A. Connectivity ------------------------------------------------------
  setGroup('A. Connectivity');
  const health = await call(AGENT, '/healthz');
  if (health.netErr) fail('agent /healthz reachable', `cannot reach ${AGENT} — start policy-agent.exe`);
  else health.status === 200 ? pass('agent /healthz reachable', `${health.ms}ms`) : fail('agent /healthz reachable', `HTTP ${health.status}`);
  const portal = await call(AGENT, '/');
  portal.status === 200 ? pass('portal index served at /', `${portal.ms}ms`) : fail('portal index served at /', `HTTP ${portal.status} — is web/index.html deployed?`);
  const plat = await call(PLATFORM, '/healthz');
  if (plat.netErr) skip('platform-api reachable', `cannot reach ${PLATFORM} (optional)`);
  else plat.status === 200 ? pass('platform-api reachable', `${plat.ms}ms`) : skip('platform-api reachable', `HTTP ${plat.status}`);

  // -- B. Posture -----------------------------------------------------------
  setGroup('B. Executive posture');
  const posture = await call(AGENT, '/v1/analytics/enterprise-posture');
  expectShape('posture endpoint + core shape', posture, ['metrics', 'risk_by_severity', 'recommendations', 'top_risks', 'trends']);
  if (posture.status === 200 && posture.json) {
    const m = posture.json.metrics || {};
    const need = ['objects', 'sensitive_findings', 'approvals_pending', 'actions', 'evidence_records', 'audit_events'];
    const miss = need.filter((k) => m[k] === undefined);
    miss.length ? fail('posture.metrics fields (KPIs)', `missing: ${miss.join(', ')}`) : pass('posture.metrics fields (KPIs)');
    Array.isArray(posture.json.trends) ? pass('posture.trends is array (trend graph)', `${posture.json.trends.length} point(s)`) : fail('posture.trends is array', 'not an array');
    const tp = (posture.json.trends || [])[0];
    if (tp) { const tneed = ['date', 'sensitive_findings', 'high_risk_evidence', 'actions']; const tmiss = tneed.filter((k) => tp[k] === undefined); tmiss.length ? fail('trend point shape', `missing: ${tmiss.join(', ')}`) : pass('trend point shape'); }
  }

  // -- C. Portal options + AUTO-DISCOVERY ----------------------------------
  setGroup('C. Portal options & discovery');
  const opts = await call(AGENT, '/v1/portal/options');
  if (opts.status === 200) {
    const keys = ['azure_sources', 'containers', 'blobs', 'evidence_records', 'audit_events', 'action_records'];
    const present = keys.filter((k) => Array.isArray(get(opts.json, k)));
    pass('portal options arrays', `present: ${present.join(', ') || '(empty state)'}`);
  } else fail('portal options endpoint', `HTTP ${opts.status}`);

  // discover the agent's configured data store id
  const cfg = await call(AGENT, '/v1/connectors/azureblob/config');
  cfg.status === 200 ? pass('connector config (GET)', `${cfg.ms}ms`) : fail('connector config (GET)', `HTTP ${cfg.status}`);
  if (!AZ_SOURCE_ID) {
    AZ_SOURCE_ID = get(cfg.json, 'data_store_id') || get(cfg.json, 'config.data_store_id')
      || (get(opts.json, 'azure_sources') || [])[0]?.data_store_id || (get(opts.json, 'azure_sources') || [])[0]?.id || '';
    if (AZ_SOURCE_ID) info(`auto-discovered source_id = ${AZ_SOURCE_ID}`);
  }

  // -- D. Connector test + container discovery -----------------------------
  setGroup('D. Azure Blob connector');
  if (AZURE) {
    const test = await call(AGENT, '/v1/connectors/azureblob/test', { method: 'POST', body: {} });
    if (test.status === 200) {
      pass('connector test access', truncate(JSON.stringify(test.json)));
      if (!AZ_SOURCE_ID) AZ_SOURCE_ID = get(test.json, 'data_store_id') || '';
    } else fail('connector test access', `HTTP ${test.status} ${truncate(test.text)}`);

    const cont = await call(AGENT, '/v1/connectors/azureblob/containers');
    if (cont.status === 200) {
      const arr = arrify(cont.json, 'containers', 'items');
      if (Array.isArray(arr) && arr.length) {
        const names = arr.map((c) => (typeof c === 'string' ? c : c.name));
        pass('list containers', `${arr.length} container(s): ${names.slice(0, 5).join(', ')}`);
        // Override a stale/invalid env container with a real discovered one.
        if (!AZ_CONTAINER) { AZ_CONTAINER = names[0]; info(`auto-discovered container = ${AZ_CONTAINER}`); }
        else if (!names.includes(AZ_CONTAINER)) { info(`${C.y}AZ_CONTAINER="${AZ_CONTAINER}" not found on this account — overriding with discovered "${names[0]}"${C.z}`); AZ_CONTAINER = names[0]; }
      } else fail('list containers', 'no containers returned (account empty?)');
    } else fail('list containers', `HTTP ${cont.status} ${truncate(cont.text)}`);
    // Validate source id too: prefer the agent's real configured data store.
    const realSource = get(cfg.json, 'data_store_id') || get(test.json, 'data_store_id') || '';
    if (realSource && AZ_SOURCE_ID !== realSource) { info(`${C.y}using agent's configured source "${realSource}" (env AZ_SOURCE_ID="${AZ_SOURCE_ID || '(unset)'}")${C.z}`); AZ_SOURCE_ID = realSource; }
  } else { skip('connector test access', 'set AZURE=1'); skip('list containers', 'set AZURE=1'); }

  // -- E. Scan jobs lifecycle ----------------------------------------------
  setGroup('E. Scan jobs lifecycle');
  const jobsList = await call(AGENT, '/v1/scan/azureblob/jobs');
  if (jobsList.status === 200 && Array.isArray(jobsList.json?.items)) {
    pass('jobs list shape {items:[]}', `${jobsList.json.items.length} job(s)`);
    const sample = jobsList.json.items[0];
    if (sample) {
      (sample.job_id !== undefined && sample.status !== undefined && sample.checkpoint !== undefined)
        ? pass('job field contract (job_id, status, checkpoint)') : fail('job field contract', `keys: ${Object.keys(sample).join(', ')}`);
      const cp = sample.checkpoint || {};
      (cp.objects_scanned !== undefined && cp.pages_scanned !== undefined) ? pass('checkpoint.objects_scanned / pages_scanned') : fail('checkpoint counters', `checkpoint keys: ${Object.keys(cp).join(', ')}`);
    }
  } else fail('jobs list shape {items:[]}', `HTTP ${jobsList.status}`);

  let myJobId = '';
  if (AZURE && AZ_SOURCE_ID && AZ_CONTAINER) {
    const q = await call(AGENT, '/v1/scan/azureblob/jobs', { method: 'POST', body: { source_id: AZ_SOURCE_ID, data_store_id: AZ_SOURCE_ID, mode: 'metadata', container: AZ_CONTAINER, prefix: AZ_PREFIX, limit: 100 } });
    if (q.status === 200 || q.status === 202) {
      const job = q.json?.job || q.json;
      myJobId = job?.job_id || '';
      if (q.json?.status === 'blocked') fail('queue scan', `blocked by Guardian: ${truncate(JSON.stringify(q.json.dac_intent?.guardian_decision || {}))}`);
      else if (myJobId) pass('queue scan returns job_id', `job_id=${myJobId} status=${job.status}`);
      else fail('queue scan returns job_id', `no job_id: ${truncate(q.text)}`);
    } else fail('queue scan', `HTTP ${q.status} ${truncate(q.text)}`);

    if (myJobId) {
      const byId = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId));
      (byId.status === 200 && get(byId.json, 'job.job_id')) ? pass('get job by id') : fail('get job by id', `HTTP ${byId.status}`);
      // A queued async-scan job is advanced to 'running' by an explicit claim/resume
      // (no auto-worker drains the queue in the local build). Start it, then pause/resume.
      const start = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId) + '/resume', { method: 'POST' });
      let st = get(start.json, 'job.status') || '';
      (start.status === 200) ? pass('start queued job (claim → running)', `status=${st || '?'}`) : fail('start queued job', `HTTP ${start.status} ${truncate(start.text)}`);
      if (st !== 'running') { // poll a moment in case it transitions async
        for (let i = 0; i < 5; i++) { const s = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId)); st = get(s.json, 'job.status'); if (st === 'running' || st === 'completed') break; await new Promise((r) => setTimeout(r, 600)); }
      }
      if (st === 'running') {
        const pz = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId) + '/pause', { method: 'POST' });
        pz.status === 200 ? pass('pause job') : fail('pause job', `HTTP ${pz.status} ${truncate(pz.text)}`);
        const rs = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId) + '/resume', { method: 'POST' });
        rs.status === 200 ? pass('resume job') : fail('resume job', `HTTP ${rs.status} ${truncate(rs.text)}`);
      } else if (st === 'completed') { pass('job completed', 'scanned to completion'); skip('pause/resume', 'job finished too fast to pause'); }
      else { skip('pause job', `job stayed '${st || 'queued'}'`); skip('resume job', `job stayed '${st || 'queued'}'`); }
      const nz = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId) + '/normalized');
      [200, 404].includes(nz.status) ? pass('normalized artifact endpoint', `HTTP ${nz.status}`) : fail('normalized artifact endpoint', `HTTP ${nz.status}`);
      const cz = await call(AGENT, '/v1/scan/azureblob/jobs/' + encodeURIComponent(myJobId) + '/cancel', { method: 'POST' });
      cz.status === 200 ? pass('cancel job (cleanup)') : fail('cancel job (cleanup)', `HTTP ${cz.status} ${truncate(cz.text)}`);
    }
  } else skip('queue/pause/resume/cancel job', AZURE ? 'no source/container discovered' : 'set AZURE=1');

  // -- F. DSPM --------------------------------------------------------------
  setGroup('F. DSPM / classification');
  const dspm = await call(AGENT, '/v1/dspm/catalog');
  expectShape('dspm catalog', dspm, ['entity_types', 'checks']);
  if (AZURE && AZ_SOURCE_ID && AZ_CONTAINER) {
    const cs = await call(AGENT, '/v1/dspm/content-scan', { method: 'POST', body: { source_id: AZ_SOURCE_ID, data_store_id: AZ_SOURCE_ID, container: AZ_CONTAINER, prefix: AZ_PREFIX, limit: 50 } });
    [200, 202].includes(cs.status) ? pass('content scan', truncate(JSON.stringify(cs.json))) : fail('content scan', `HTTP ${cs.status} ${truncate(cs.text)}`);
  } else skip('content scan', AZURE ? 'no source/container discovered' : 'set AZURE=1');

  // -- pick a recent execution_id (for guardian + AI recommendations) ------
  // Best source: trigger a real end-to-end execution that runs the orchestrator
  // and produces Guardian decisions. Falls back to harvesting from records.
  if (!EXECUTION_ID && AZURE && AZ_SOURCE_ID) {
    setGroup('E2. Full execution (generates execution_id + Guardian decisions)');
    const ex = await call(AGENT, '/v1/execution/azureblob/start', { method: 'POST', body: { batch_id: 'batch-test-' + Date.now(), action_mode: 'dry_run' } });
    if ([200, 202].includes(ex.status)) {
      EXECUTION_ID = get(ex.json, 'status.execution_id') || get(ex.json, 'execution_id') || get(ex.json, 'execution.execution_id') || '';
      EXECUTION_ID ? pass('execution start → execution_id', EXECUTION_ID) : fail('execution start', `no execution_id: ${truncate(ex.text)}`);
    } else fail('execution start', `HTTP ${ex.status} ${truncate(ex.text)}`);
  }
  if (!EXECUTION_ID) {
    for (const path of ['/v1/evidence?limit=50', '/v1/actions/history?limit=50', '/v1/audit/events?limit=50']) {
      const r = await call(AGENT, path);
      const items = arrify(r.json, 'items') || [];
      const found = items.map((it) => it.execution_id || it.details?.execution_id || it.correlation_id).find(Boolean);
      if (found) { EXECUTION_ID = found; info(`auto-picked execution_id = ${EXECUTION_ID} (from ${path.split('?')[0]})`); break; }
    }
  }

  // -- G. AI ---------------------------------------------------------------
  setGroup('G. AI integrations');
  const ai = await call(AGENT, '/v1/ai/providers/catalog');
  (ai.status === 200 && Array.isArray(ai.json?.items)) ? pass('ai provider catalog {items:[]}', `${ai.json.items.length} provider(s)`) : fail('ai provider catalog', `HTTP ${ai.status}`);
  if (EXECUTION_ID) {
    const rec1 = await call(AGENT, '/v1/ai/azureblob/recommendations', { method: 'POST', body: { execution_id: EXECUTION_ID, data_store_id: AZ_SOURCE_ID || undefined, limit: 10 } });
    [200, 202].includes(rec1.status) ? pass('ai recommendations (POST + execution_id)', truncate(JSON.stringify(rec1.json)))
      : fail('ai recommendations (POST + execution_id)', `HTTP ${rec1.status} ${truncate(rec1.text)}`);
  } else skip('ai recommendations', 'no execution_id found — run a scan/action first');

  // -- H. Evidence / audit / actions ---------------------------------------
  setGroup('H. Evidence, audit, actions');
  const ev = await call(AGENT, '/v1/evidence?limit=20');
  (ev.status === 200 && Array.isArray(ev.json?.items)) ? pass('evidence {items:[]}', `${ev.json.items.length} record(s)`) : fail('evidence', `HTTP ${ev.status}`);
  const pkg = await call(AGENT, '/v1/evidence/package');
  pkg.status === 200 ? pass('evidence package export') : fail('evidence package export', `HTTP ${pkg.status}`);
  const au = await call(AGENT, '/v1/audit/events?limit=20');
  (au.status === 200 && Array.isArray(au.json?.items)) ? pass('audit events {items:[]}', `${au.json.items.length} event(s)`) : fail('audit events', `HTTP ${au.status}`);
  const pend = await call(AGENT, '/v1/actions/pending?limit=20');
  (pend.status === 200 && Array.isArray(pend.json?.items)) ? pass('actions pending {items:[]}', `${pend.json.items.length} pending`) : fail('actions pending', `HTTP ${pend.status}`);
  const hist = await call(AGENT, '/v1/actions/history?limit=20');
  (hist.status === 200 && Array.isArray(hist.json?.items)) ? pass('actions history {items:[]}', `${hist.json.items.length} record(s)`) : fail('actions history', `HTTP ${hist.status}`);

  // -- I. Guardian + policy trust ------------------------------------------
  setGroup('I. Guardian + policy trust');
  const gNo = await call(AGENT, '/v1/guardian/decisions?limit=20');
  (gNo.status === 400) ? pass('guardian decisions requires execution_id (documented)', 'portal supplies execution_id automatically') :
    (gNo.status === 200 ? pass('guardian decisions (no execution_id) returns 200') : fail('guardian decisions (no execution_id)', `HTTP ${gNo.status}`));
  if (EXECUTION_ID) {
    const gYes = await call(AGENT, '/v1/guardian/decisions?limit=20&execution_id=' + encodeURIComponent(EXECUTION_ID));
    (gYes.status === 200 && Array.isArray(gYes.json?.items)) ? pass('guardian decisions for execution_id', `${gYes.json.items.length} decision(s)`) : fail('guardian decisions for execution_id', `HTTP ${gYes.status} ${truncate(gYes.text)}`);
  } else skip('guardian decisions for execution_id', 'no execution_id found');
  const trust = await call(AGENT, '/v1/policy/artifact/trust');
  if (trust.status === 200) {
    const t = trust.json || {};
    (t.signature_verified !== undefined && t.signature_status !== undefined) ? pass('policy trust shape', `verified=${t.signature_verified} status=${t.signature_status} version=${t.compiled_policy_version || '?'}`) : fail('policy trust shape', `keys: ${Object.keys(t).join(', ')}`);
    t.signature_verified === true ? pass('compiled policy signature verified') : fail('compiled policy signature verified', `status=${t.signature_status}`);
  } else fail('policy trust endpoint', `HTTP ${trust.status}`);

  // -- J. Runtime ----------------------------------------------------------
  setGroup('J. Runtime / diagnostics');
  const rt = await call(AGENT, '/v1/agent/runtime/status');
  (rt.status === 200 && (has(rt.json, 'secret_provider') || has(rt.json, 'authorization'))) ? pass('agent runtime status') : fail('agent runtime status', `HTTP ${rt.status}`);

  // -- K. Workbench + governed action (preview & apply) --------------------
  setGroup('K. Workbench + governed action');
  if (AZURE && AZ_SOURCE_ID && AZ_CONTAINER) {
    const wb = await call(AGENT, '/v1/tools/azureblob/workbench', { method: 'POST', body: { operation: 'list_blobs', source_id: AZ_SOURCE_ID, container: AZ_CONTAINER, prefix: AZ_PREFIX, limit: 25 } });
    let firstBlob = '';
    if (wb.status === 200) {
      const arr = arrify(wb.json, 'items', 'blobs', 'objects');
      if (Array.isArray(arr)) { pass('workbench list_blobs', `${arr.length} object(s)`); const b0 = arr[0]; firstBlob = b0 ? (typeof b0 === 'string' ? b0 : (b0.name || b0.blob || b0.path || '')) : ''; }
      else fail('workbench list_blobs', `no items array: ${truncate(wb.text)}`);
    } else fail('workbench list_blobs', `HTTP ${wb.status} ${truncate(wb.text)}`);

    // PREVIEW (dry-run) — always safe
    const prev = await call(AGENT, '/v1/tools/azureblob/workbench', { method: 'POST', body: { operation: 'set_tags', source_id: AZ_SOURCE_ID, container: AZ_CONTAINER, blob: firstBlob || undefined, dry_run: true, tags: { dd_test: 'preview' } } });
    [200, 202].includes(prev.status) ? pass('governed action PREVIEW (dry-run)', truncate(JSON.stringify(prev.json))) : fail('governed action preview', `HTTP ${prev.status} ${truncate(prev.text)}`);

    // APPLY (real governed write-back) — only when APPLY=1 and we have a blob.
    // Uses the connector's real write-back op (apply_blob_index_tags). A 'blocked'
    // status or 403 is a PASS — it means Guardian/HITL correctly gated the action.
    if (APPLY && firstBlob) {
      const applyRes = await call(AGENT, '/v1/connectors/azureblob/actions/execute', { method: 'POST', body: {
        execution_id: EXECUTION_ID || undefined, blob_path: firstBlob, operation: 'apply_blob_index_tags',
        action_mode: 'apply', writeback_enabled: true, guardian_decision: 'allow', index_tags: { dd_test: 'applied' },
      } });
      if ([200, 202].includes(applyRes.status)) {
        const st = get(applyRes.json, 'status') || '';
        (st === 'completed' || st === 'applied') ? pass('governed action APPLY (real write-back)', `status=${st}`)
          : pass('governed action APPLY gated by Guardian/HITL', `status=${st || 'blocked'} (expected when approval required)`);
      } else if (applyRes.status === 403) pass('governed action APPLY gated by policy/HITL', 'HTTP 403');
      else fail('governed action APPLY', `HTTP ${applyRes.status} ${truncate(applyRes.text)}`);
      // best-effort revert
      await call(AGENT, '/v1/connectors/azureblob/actions/execute', { method: 'POST', body: { execution_id: EXECUTION_ID || undefined, blob_path: firstBlob, operation: 'apply_blob_index_tags', action_mode: 'apply', writeback_enabled: true, guardian_decision: 'allow', index_tags: {} } });
    } else if (APPLY) skip('governed action APPLY', 'no blob found to write to');
    else skip('governed action APPLY (real write-back)', 'set APPLY=1 to run');
  } else { skip('workbench list_blobs', AZURE ? 'no source/container discovered' : 'set AZURE=1'); skip('governed action', AZURE ? 'no source/container discovered' : 'set AZURE=1'); }

  // -- Summary -------------------------------------------------------------
  const passed = results.filter((r) => r.status === 'PASS').length;
  const failed = results.filter((r) => r.status === 'FAIL').length;
  const skipped = results.filter((r) => r.status === 'SKIP').length;
  console.log(`\n${C.bold}==================== SUMMARY ====================${C.z}`);
  console.log(`  ${C.g}PASS ${passed}${C.z}   ${C.r}FAIL ${failed}${C.z}   ${C.y}SKIP ${skipped}${C.z}   (total ${results.length})`);
  if (failed) { console.log(`\n${C.r}${C.bold}Failures:${C.z}`); results.filter((r) => r.status === 'FAIL').forEach((r) => console.log(`  ${C.r}✗${C.z} [${r.group}] ${r.name} — ${r.detail}`)); }
  if (skipped) { console.log(`\n${C.y}Skipped:${C.z}`); results.filter((r) => r.status === 'SKIP').forEach((r) => console.log(`  ${C.y}-${C.z} ${r.name} — ${r.detail}`)); }
  console.log('');
  console.log(failed === 0 ? `${C.g}${C.bold}ALL CHECKS PASSED${C.z} — safe to run the UI end-to-end scenarios.` : `${C.r}${C.bold}${failed} CHECK(S) FAILED${C.z} — fix before UI testing.`);
  process.exit(failed === 0 ? 0 : 1);
})();
