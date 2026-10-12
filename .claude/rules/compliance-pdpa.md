# PDPA / Personal Data Protection (mandatory)

We process personal data of users in Southeast Asia, so the code must satisfy
the regional personal-data laws. They share the same core obligations; build to
the **strictest** of them so one implementation serves every market.

| Jurisdiction | Law | Regulator |
|--------------|-----|-----------|
| Singapore | Personal Data Protection Act 2012 (PDPA, amended 2020) | PDPC |
| Indonesia | UU No. 27/2022 Pelindungan Data Pribadi (UU PDP) | PDP agency (Lembaga PDP) |
| Malaysia | Personal Data Protection Act 2010 (amended 2024) | JPDP / PDP Commissioner |
| Thailand | Personal Data Protection Act B.E. 2562 (2019) | PDPC Thailand |

> This rule translates legal obligations into engineering requirements. It is not
> legal advice — product/legal decide the lawful basis, notice text, and DPO;
> engineering makes sure the system can honor them. Data classes are defined in
> [data-classification](data-classification.md).

## 1. Lawful basis & consent

- Every collection of PII has a recorded **purpose** and **lawful basis**
  (consent, contract, legal obligation, legitimate interest…). A new PII field
  in a proto/migration needs that stated in the PR.
- **Consent is recorded, not assumed:** store `who`, `what purpose`, `notice
  version`, `when`, `how` (UI surface) in a consent table. No pre-ticked boxes.
- **Withdrawal** must be as easy as giving consent and must take effect in
  processing (stop marketing sends, stop analytics, stop third-party sharing).
- **Restricted PII** (health, biometric, financial, children, government ID —
  "specific personal data" under UU PDP / "sensitive" under TH & MY 2024) needs
  explicit consent and a data-protection impact assessment (DPIA) before launch.
- **Children:** under-age users need guardian consent (UU PDP, TH PDPA); gate
  sign-up flows accordingly.

## 2. Purpose limitation & data minimization

- Collect only fields the stated purpose needs. Optional fields are optional in
  the API too (`phone` in `RegisterReq` is not required — keep it that way).
- Don't reuse data for a new purpose (e.g. sending support-ticket text to an LLM
  for model improvement) without a new basis/consent.
- Don't add PII to analytics events, logs, traces, or metric labels — use
  `user_id` (see [observability](observability.md)).

## 3. Data-subject rights — the system must support these

| Right | SG | ID | MY | TH | Engineering requirement |
|-------|----|----|----|----|------------------------|
| Access | ✓ | ✓ | ✓ | ✓ | Export a user's data (all tables + derived stores) in machine-readable JSON |
| Correction | ✓ | ✓ | ✓ | ✓ | User-editable profile; admin correction is audited |
| Erasure / deletion | (retention limit) | ✓ | (retention limit) | ✓ | Hard-delete or irreversible anonymization pipeline (see §6) |
| Withdraw consent | ✓ | ✓ | ✓ | ✓ | Consent flags honored by every consumer |
| Portability | enacted, not yet in force | ✓ | ✓ (2024) | ✓ | Same export, structured format |
| Object / restrict processing | – | ✓ | ✓ (likely damage, direct marketing) | ✓ | Per-purpose processing flags |

- Requests are tracked (ticket ID), identity-verified, completed within the
  statutory window (SG access: **30 days**; ID: **3×24 h** for several rights
  per UU PDP; TH: **30 days**) and **audit-logged** ([compliance-soc2](compliance-soc2.md) CC7).
- Erasure must reach **every copy**: Postgres, Redis cache, search indexes,
  RabbitMQ dead-letter queues, object storage, the `apps/ai` vector store / LLM
  conversation memory, and analytics. Backups age out within their retention
  window (document it).

## 4. Protection obligation

Reasonable security arrangements = everything in [security](security.md),
[api-security](api-security.md), [mobile-security](mobile-security.md), and
[compliance-soc2](compliance-soc2.md): encryption in transit/at rest, access
control, audit logging, secure SDLC. Non-prod environments never hold real
personal data — use synthetic seeds (`make seed`) or masked copies.

## 5. Cross-border transfer

- Data may leave the country of collection only to destinations/vendors with
  comparable protection (SG: Transfer Limitation Obligation; ID: adequacy or
  safeguards or consent; TH/MY: adequacy or appropriate safeguards).
- Record **where each datastore and each subprocessor runs** (region). Adding a
  vendor in a new region — including an LLM API — is a compliance change, not
  just a config change.

## 6. Retention & disposal

- Every PII table declares a retention period in its migration header:
  ```sql
  -- data-class: PII (email, name, phone)
  -- retention: account lifetime + 30 days after deletion request, then hard delete
  ```
- Soft delete (`deleted_at`) is **not** erasure. A scheduled job (worker) must
  hard-delete or anonymize rows past retention; anonymization must be
  irreversible (no reversible hash of email/phone).
- Logs/traces containing Internal IDs follow log retention (30–90 days).

## 7. Breach notification

The clock starts when a breach is **assessed** as notifiable — engineering must
be able to answer *what, whose, how many, since when* fast:

| Jurisdiction | Notify regulator | Notify individuals |
|--------------|-----------------|--------------------|
| Singapore | within **3 calendar days** of assessing it notifiable (significant harm or ≥ 500 individuals) | as soon as practicable if significant harm |
| Indonesia | written notice within **3×24 hours** (to subjects and the agency) | same deadline |
| Malaysia | **72 hours** to the Commissioner (2024 amend.) | within 7 days if significant harm |
| Thailand | within **72 hours** of becoming aware | without delay if high risk |

Engineering prerequisites: the audit log (who accessed what), a data inventory
(which tables hold which classes), and the ability to enumerate affected
`user_id`s from an incident window.

## 8. Accountability

- Data inventory / record of processing lives in `docs/` and is updated in the
  same PR that adds a PII field, datastore, or subprocessor.
- A DPO is appointed (SG/MY/TH/ID requirement for most processors at scale) —
  link them on DPIA-required changes.
- Privacy by design: new features with PII get a short privacy review in the
  PR (*Security & compliance* section of the template).

## PR checklist (PII-touching changes)

- [ ] New PII fields classified, purpose + lawful basis stated
- [ ] Field excluded from logs/spans/metrics/analytics
- [ ] Included in the user data export and the erasure pipeline
- [ ] Retention declared in the migration header
- [ ] No new third-party/region without DPA + inventory update
