export interface IDataFeed {
	general: IGeneralInfo;
	pilots: IPilot[];
	atis: IAtis[];
	prefiles: IPrefile[];
	rating: IRating[];
	pilotRating: IPilotRating[];
	militaryRating: IMilitaryRating[];
}

export interface IGeneralInfo {
	version: number;
	reload: number;
	update: string;
	update_timestamp: Date;
	connected_clients: number;
	unique_users: number;
}

export interface IPilot {
	cid: number;
	name: string;
	callsign: string;
	server: string;
	pilot_rating: number;
	military_rating: number;
	latitude: number;
	longitude: number;
	altitude: number;
	groundspeed: number;
	transponder: string;
	heading: number;
	qnh_i_hg: number;
	qnh_mb: number;
	flight_plan?: IFlightPlan;
	logon_time: Date;
	last_updated: Date;
}

export interface IFlightPlan {
	flight_rules: string;
	aircraft: string;
	aircraft_faa: string;
	aircraft_short: string;
	departure: string;
	arrival: string;
	alternate: string;
	cruise_tas: string;
	altitude: string;
	deptime: string;
	enroute_time: string;
	fuel_time: string;
	remarks: string;
	route: string;
	revision_id: number;
	assigned_transponder: string;
}

export interface IAtis {
	cid: number;
	name: string;
	callsign: string;
	frequency: string;
	facility: number;
	rating: number;
	server: string;
	visual_range: number;
	atis_code: string;
	text_atis: string[];
	last_updated: Date;
	logon_time: Date;
}

export interface IPrefile {
	cid: number;
	name: string;
	callsign: string;
	flight_plan: IFlightPlan;
	last_updated: Date;
}

export interface IRating {
	id: number;
	short: string;
	long: string;
}

export interface IPilotRating {
	id: number;
	short_name: string;
	long_name: string;
}

export interface IMilitaryRating {
	id: number;
	short_name: string;
	long_name: string;
}
