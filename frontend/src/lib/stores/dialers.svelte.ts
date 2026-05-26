// Dialers state — number matching and flagged numbers.
// Matching logic is delegated to the Rust number-matcher worker via Go.

export interface NumberItem {
	number: string;
	hiyaNumber?: string;
	hiyaSpamLabel?: string;
	departmentName?: string;
	officeName?: string;
	assignedType?: string;
	assignmentName?: string;
	numberStatus?: string;
	reservedReason?: string;
	previousDepartmentName?: string;
	isReservedNumber: boolean;
	inNumberVerifier: boolean;
	inHiya: boolean;
	updatedAt?: string;
	lastSeenAt?: string;
}

export interface NumbersMatchData {
	items: NumberItem[];
	totalCount: number;
	matchedCount: number;
	unmatchedCount: number;
	officeCount: number;
	syncedAt: string;
	source: string;
}

export const numbersData = $state<NumbersMatchData>({
	items: [],
	totalCount: 0,
	matchedCount: 0,
	unmatchedCount: 0,
	officeCount: 0,
	syncedAt: '',
	source: 'extension',
});

export const numbersFilters = $state({
	office: 'all',
	department: 'all',
	search: '',
	view: 'all' as 'all' | 'nv_missing' | 'hiya_missing' | 'done',
});

export const flaggedData = $state({
	headers: [] as string[],
	rows: [] as Record<string, string>[],
	fileName: '',
	importedAt: '',
});

export function updateNumbersData(data: NumbersMatchData) {
	Object.assign(numbersData, data);
}

export function updateFlaggedData(data: typeof flaggedData) {
	Object.assign(flaggedData, data);
}
