export interface IPirepFeed {
	receiptTime: Date;
	obsTime: number;
	qcField: number;
	icaoId: string;
	acType: string;
	lat: number;
	lon: number;
	fltLvl: number;
	fltLvelType: 'GRND' | 'DURC' | 'DURD' | 'CRUISE' | 'OTHER' | 'UNKN';
	clouds?: {
		cover: 'CLR' | 'CAVOK' | 'FEW' | 'SCT' | 'BKN' | 'OVC' | 'OVX';
		base: number;
		top: number;
	}[];
	visib?: string;
	wxString: string;
	temp?: number;
	wdir?: number;
	wspd?: number;
	icgBas1?: number;
	icgTop1?: number;
	icgInt1:
		| 'NEG'
		| 'NEGclr'
		| 'TRC'
		| 'TRC-LGT'
		| 'LGT'
		| 'LGT-MOD'
		| 'MOD'
		| 'MOD-SEV'
		| 'HVY'
		| 'SEV';
	icgType1: 'RIME' | 'CLEAR' | 'MIXED';
	icgBas2?: number;
	icgTop2?: number;
	icgInt2:
		| 'NEG'
		| 'NEGclr'
		| 'TRC'
		| 'TRC-LGT'
		| 'LGT'
		| 'LGT-MOD'
		| 'MOD'
		| 'MOD-SEV'
		| 'HVY'
		| 'SEV';
	icgType2: 'RIME' | 'CLEAR' | 'MIXED';
	tbBas1?: number;
	tbTop1?: number;
	tbInt1:
		| 'NEG'
		| 'SMTH-LGT'
		| 'LGT'
		| 'LGT-MOD'
		| 'MOD'
		| 'MOD-SEV'
		| 'SEV'
		| 'MOD-EXTM'
		| 'SEV-EXTM'
		| 'EXTM';
	tbType1: 'CAT' | 'CHOP' | 'LLWS' | 'MWAVE';
	tbFreq1: 'ISOL' | 'OCNL' | 'CONT';
	tbBas2?: number;
	tbTop2?: number;
	tbInt2:
		| 'NEG'
		| 'SMTH-LGT'
		| 'LGT'
		| 'LGT-MOD'
		| 'MOD'
		| 'MOD-SEV'
		| 'SEV'
		| 'MOD-EXTM'
		| 'SEV-EXTM'
		| 'EXTM';
	tbType2: 'CAT' | 'CHOP' | 'LLWS' | 'MWAVE';
	tbFreq2: 'ISOL' | 'OCNL' | 'CONT';
	vertGust?: number;
	brkAction: 'GOOD' | 'GOOD-MED' | 'MED' | 'MED-POOR' | 'POOR' | 'NIL';
	pirepType: 'PIREP' | 'Urgent PIREP' | 'AIREP' | 'AMDAR';
	rawOb: string;
}
