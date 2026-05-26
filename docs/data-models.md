# Data Models

TypeScript-annotated shapes for all major state objects in the GHL Manager.

---

## Storage Keys (localStorage)

```typescript
const STORAGE_KEY          = 'ghl_loc_tokens_v2';          // { [locationId]: string }
const DOMAIN_KEY           = 'ghl_loc_domains';             // { [locationId]: string }
const ACUITY_KEY           = 'ghl_loc_acuity_fields';       // { [locationId]: string }
const CALENDAR_IDS_KEY     = 'ghl_loc_calendar_ids';        // { [locationId]: string }
const ACCOUNTS_LIBRARY_KEY = 'ghl_accounts_library_v1';    // { [locationId]: AccountLibraryEntry }
const FLAGGED_STORAGE_KEY  = 'ghl_flagged_numbers_v1';     // FlaggedData
const FLAGGED_INBOX_KEY    = 'ghl_flagged_numbers_inbox_v1'; // pending from extension
const GHL_NUMBERS_STORAGE_KEY = 'ghl_numbers_match_v1';    // NumbersMatchData
const GHL_NUMBERS_INBOX_KEY   = 'ghl_numbers_match_inbox_v1'; // pending from extension
const GHL_NUMBERS_LIBRARY_KEY = 'ghl_numbers_library_v1';  // { [normalizedNumber]: NumberLibraryEntry }
const GHL_FORM_AUTH_INBOX_KEY = 'ghl_form_auth_inbox_v1';  // form session tokens from extension
const FORM_SESSION_TOKEN_KEY  = 'ghl_form_session_tokens_v1'; // { [locId]: string[] }
// Legacy (still read at startup):
// 'ghl_agency_token' — raw agency token string
// 'ghl_company_id'   — raw company ID string
```

---

## Core Session State (globals)

```typescript
let agencyToken: string = '';       // pit-... agency-level token
let companyId: string = '';         // 24-char MongoDB ObjectID
let allLocs: Location[] = [];       // all sub-accounts from /locations/search
let selected: Set<string> = new Set(); // locationIds selected by user
let cvData: Record<string, LocationCVData> = {};
let allCvKeys: string[] = [];       // sorted union of all CV names across selected accounts
```

---

## Location (sub-account)

```typescript
interface Location {
  id: string;             // e.g. "abc123defghi"
  name: string;           // display name
  businessName?: string;  // fallback if name is missing
  email?: string;
  phone?: string;
  address?: string;
  city?: string;
  state?: string;
  country?: string;
  postalCode?: string;
}
```

---

## AccountLibraryEntry

Persisted per-location metadata, supplementing what GHL returns.

```typescript
interface AccountLibraryEntry {
  name: string;           // location name (snapshot)
  token: string;          // sub-account pit-... token
  domain: string;         // e.g. "no-needle-needed.com"
  acuity: string;         // e.g. "field:12345678"
  calendarIds: string;    // comma-separated GHL calendar IDs
  active: boolean;        // include in bulk operations
}
// Keyed by locationId: Record<string, AccountLibraryEntry>
```

---

## LocationCVData

```typescript
interface CVItem {
  id: string;
  name: string;
  value: string;
  fieldType?: string;
}

interface LocationCVData {
  name: string;           // location display name
  token: string;          // the token used to load this data
  cvs: CVItem[];
  forms?: FormItem[];
  triggerLinks?: TriggerLinkItem[];
  error?: string;         // set if the fetch failed
}

interface FormItem {
  id: string;
  name: string;
  submissionWebhook?: string;
}

interface TriggerLinkItem {
  id: string;
  name: string;
  link: string;
}
```

---

## FlaggedData

```typescript
interface FlaggedData {
  headers: string[];                      // CSV column headers
  rows: Record<string, string>[];         // array of { [header]: value }
  fileName: string;                       // original CSV filename
  importedAt: string;                     // locale string timestamp
}
// Defaults: { headers: [], rows: [], fileName: '', importedAt: '' }
```

---

## NumbersMatchData

```typescript
interface NumberItem {
  number: string;                 // E.164 phone number, e.g. "+15555551234"
  hiyaNumber?: string;            // Hiya-registered number (may differ)
  hiyaSpamLabel?: string;
  departmentName?: string;
  officeName?: string;
  assignedType?: string;          // e.g. "user", "department"
  assignmentName?: string;        // name of user or department assigned to
  numberStatus?: string;
  reservedReason?: string;
  previousDepartmentName?: string;
  isReservedNumber: boolean;
  inNumberVerifier: boolean;      // registered with Number Verifier
  inHiya: boolean;                // registered with Hiya
  nvStatus?: { success: boolean; message?: string };
  hiyaStatus?: { success: boolean; message?: string };
  updatedAt?: string;             // ISO timestamp
  lastSeenAt?: string;            // ISO timestamp
}

interface NumbersMatchData {
  items: NumberItem[];
  totalCount: number;
  matchedCount: number;           // count where inNumberVerifier === true
  unmatchedCount: number;
  officeCount: number;
  syncedAt: string;               // ISO timestamp
  source: 'extension' | string;
}
// Defaults: { items: [], totalCount: 0, matchedCount: 0, unmatchedCount: 0, officeCount: 0, syncedAt: '', source: 'extension' }
```

---

## NumberLibraryEntry

Persisted per-number metadata (survives across extension syncs).

```typescript
interface NumberLibraryEntry {
  number: string;
  nv: boolean;                    // confirmed in Number Verifier
  hiya: boolean;                  // confirmed in Hiya
  hiyaSpamLabel?: string;
  hiyaNumber?: string;
  officeName?: string;
  departmentName?: string;
  assignedType?: string;
  assignmentName?: string;
  numberStatus?: string;
  reservedReason?: string;
  previousDepartmentName?: string;
  isReservedNumber: boolean;
  lastSeenAt: string;             // ISO timestamp
  updatedAt: string;              // ISO timestamp
}
// Keyed by compactPhone(number): Record<string, NumberLibraryEntry>
// compactPhone() strips all non-digits: "+1 (555) 555-1234" → "15555551234"
```

---

## NumbersRegistrationState (in-memory only)

```typescript
type RegistrationStatus = 'pending' | 'done' | 'error';

interface NumberRegistrationEntry {
  nv?: RegistrationStatus;
  hiya?: RegistrationStatus;
}
// Keyed by compactPhone(number)
let numbersRegistrationState: Record<string, NumberRegistrationEntry> = {};
```

---

## apiFetch Return Shapes

```typescript
// apiFetch — throws on non-ok
async function apiFetch(token: string, method: string, path: string, body?: object): Promise<any>

// apiFetchDetailed — never throws, returns status
interface DetailedResponse {
  ok: boolean;
  status: number;
  text: string;
  json: any | null;
}
```

---

## Meta Pixel Brands

```typescript
interface PixelBrand {
  id: string;           // numeric Facebook pixel ID
  label: string;        // display name
}

const PIXEL_BRANDS: PixelBrand[] = [
  { id: '1180664739949233', label: 'First Touch Beauty' },
  { id: '540031443835405',  label: 'No Needle Needed' },
  { id: '819221296693583',  label: 'Advanced Beauty Treatments' },
];
```
