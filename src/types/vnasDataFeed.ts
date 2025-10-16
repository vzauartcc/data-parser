export interface IControllerFeed {
	updatedAt: Date;
	controllers: IController[];
}

export interface IController {
	artccId: string;
	primaryFacilityId: string;
	primaryPositionId: string;
	role: string;
	positions: IPosition[];
	isActive: boolean;
	isObserver: boolean;
	loginTime: Date;
	vatsimData: {
		cid: string;
		realName: string;
		controllerInfo: string;
		userRating: string;
		requestedRating: string;
		callsign: string;
		facilityType: string;
		primaryFrequency: number;
	};
}

export interface IPosition {
	facility: string;
	facilityName: string;
	positionName: string;
	positionType: string;
	radioName: string;
	defaultCallsign: string;
	frequency: number;
	isPrimary: boolean;
	isActive: true;
	eramData: {
		sectorId: string;
	};
	starsData: {
		subset: number;
		sectorId: string;
		areaId: string;
	};
}
