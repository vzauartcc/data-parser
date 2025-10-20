export interface IControllerFeed {
	updatedAt: Date;
	controllers: IController[];
}

export interface IController {
	artccId: string;
	primaryFacilityId: string;
	primaryPositionId: string;
	role: 'Observer' | 'Controller' | 'Student' | 'Instructor';
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
		facilityType:
			| 'Observer'
			| 'FlightServiceStation'
			| 'ClearanceDelivery'
			| 'Ground'
			| 'Tower'
			| 'ApproachDeparture'
			| 'Center';
		primaryFrequency: number;
	};
}

export interface IPosition {
	facility: string;
	facilityName: string;
	positionName: string;
	positionType: 'Artcc' | 'Tracon' | 'Atct';
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
