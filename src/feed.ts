import { Redis } from 'ioredis';
import { airports, airspace, neighbors, vNasRatings } from './constants.js';
import { AtcOnlineModel } from './models/atcOnline.js';
import { ControllerHoursModel } from './models/controllerHours.js';
import { PilotOnlineModel } from './models/pilotOnline.js';
import { PirepModel } from './models/pirep.js';
import { UserModel } from './models/user.js';
import type { IPirepFeed } from './types/pirepFeed.js';
import type { IDataFeed } from './types/vatsimDataFeed.js';
import type { IControllerFeed } from './types/vnasDataFeed.js';

const postToZAUApi = async (uri: string) => {
	try {
		return await fetch(`${process.env['ZAU_API_URL']}/stats/fifty/${uri}`, {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${process.env['ZAU_API_KEY']}`,
				'Content-Type': 'application/json',
			},
		});
	} catch (message) {
		return console.error(message);
	}
};

export async function fetchPilots(redis: Redis) {
	try {
		await PilotOnlineModel.deleteMany({}).exec();

		fetch(`https://data.vatsim.net/v3/vatsim-data.json?t=${Date.now()}`, {
			method: 'GET',
			headers: {
				'Content-Type': 'application/json',
			},
		})
			.then((response) => response.json())
			.then(async (data) => {
				const vatsimData = data as IDataFeed;

				const dataPilots = [];
				const redisData = await redis.get('pilots');
				const redisPilots = redisData && redisData.length ? redisData.split('|') : [];

				for (const pilot of vatsimData.pilots) {
					if (
						pilot.flight_plan &&
						pilot.flight_plan.aircraft_faa &&
						pilot.flight_plan.altitude &&
						(airports.includes(pilot.flight_plan.departure) ||
							airports.includes(pilot.flight_plan.departure) ||
							isPointInPolygon([pilot.longitude, pilot.latitude]))
					) {
						PilotOnlineModel.create({
							cid: pilot.cid,
							name: pilot.name,
							callsign: pilot.callsign,
							aircraft: pilot.flight_plan.aircraft_faa,
							dep: pilot.flight_plan.departure,
							dest: pilot.flight_plan.arrival,
							code: pilot.transponder,
							lat: pilot.latitude,
							lng: pilot.longitude,
							altitude: pilot.altitude,
							heading: pilot.heading,
							speed: pilot.groundspeed,
							planned_cruise: pilot.flight_plan.altitude.includes('FL')
								? pilot.flight_plan.altitude.replace('FL', '') + '00'
								: pilot.flight_plan.altitude, // If flight plan altitude is 'FL350' instead of '35000'
							route: pilot.flight_plan.route || '',
							remakes: pilot.flight_plan.remarks || '',
						});

						dataPilots.push(pilot.callsign);

						redis.hmset(
							`PILOT:${pilot.callsign}`,
							'callsign',
							pilot.callsign,
							'lat',
							`${pilot.latitude}`,
							'lng',
							`${pilot.longitude}`,
							'speed',
							`${pilot.groundspeed}`,
							'heading',
							`${pilot.heading}`,
							'altitude',
							`${pilot.altitude}`,
							'cruise',
							`${pilot.flight_plan.altitude.includes('FL') ? pilot.flight_plan.altitude.replace('FL', '') + '00' : pilot.flight_plan.altitude}`,
							'destination',
							`${pilot.flight_plan.arrival}`,
						);
						redis.expire(`PILOT:${pilot.callsign}`, 300);
						redis.publish(`PILOT:UPDATE`, pilot.callsign);
					}
				}

				for (const pilot of redisPilots) {
					if (!dataPilots.includes(pilot)) {
						redis.publish('PILOT:DELETE', pilot);
					}
				}

				redis.set('pilots', dataPilots.join('|'));
				redis.expire('pilots', 65);
			})
			.catch((err) => console.log('Error doing VATSIM datafeed fetch:', err));
	} catch (err) {
		console.log('Error updating pilots', err);
	}
}

export async function fetchControllers(redis: Redis) {
	try {
		await AtcOnlineModel.deleteMany({}).exec();

		fetch(`https://live.env.vnas.vatsim.net/data-feed/controllers.json?t=${Date.now()}`)
			.then((response) => response.json())
			.then(async (data) => {
				const vatsimData = data as IControllerFeed;

				const dataControllers = [];
				const redController = await redis.get('controllers');
				const redisControllers =
					redController && redController.length ? redController.split('|') : [];

				const dataNeighbors = [];

				for (const controller of vatsimData.controllers) {
					if (controller.artccId === 'ZAU') {
						const user = await UserModel.findOne({
							cid: parseInt(controller.vatsimData.cid, 10),
						}).exec();
						const controllerName = user
							? `${user.fname} ${user.lname}`
							: controller.vatsimData.realName;

						if (controller.isActive) {
							AtcOnlineModel.create({
								cid: controller.vatsimData.cid,
								name: controllerName,
								rating: vNasRatings[controller.vatsimData.requestedRating],
								pos: controller.vatsimData.callsign,
								timeStart: controller.loginTime,
								atis: controller.vatsimData.controllerInfo,
								frequency: controller.vatsimData.primaryFrequency,
							});

							dataControllers.push(controller.vatsimData.callsign);
						}

						const session = await ControllerHoursModel.findOne({
							cid: parseInt(controller.vatsimData.cid, 10),
							timeStart: controller.loginTime,
						});

						if (!session) {
							if (controller.isActive) {
								ControllerHoursModel.create({
									cid: parseInt(controller.vatsimData.cid, 10),
									timeStart: controller.loginTime,
									timeEnd: new Date(new Date().toUTCString()),
									position: controller.vatsimData.callsign,
									isStudent: controller.role === 'Student',
									isInstructor: controller.role === 'Instructor',
								});
								const rData = [
									{
										cid: parseInt(controller.vatsimData.cid, 10),
										name: controller.vatsimData.realName,
										rating: vNasRatings[controller.vatsimData.requestedRating],
										pos: controller.vatsimData.callsign,
										timeStart: controller.loginTime,
										atis: controller.vatsimData.controllerInfo,
										frequency: controller.vatsimData.primaryFrequency,
									},
								];

								redis.lpush('myQueue', JSON.stringify(rData), (error) => {
									if (error) {
										console.log('Redis LPUSH error:', error);
									}
								});

								postToZAUApi(controller.vatsimData.cid);
							}
						} else {
							session.timeEnd = new Date(new Date().toUTCString());
							if (!controller.isActive) {
								// Controller went inactive. Modify the timeStart to uniquely identify active sessions
								session.timeStart = new Date(session.timeStart.getTime() - 1000);
							}
							session.save();
						}

						redis.expire(`CONTROLLER:${controller.vatsimData.callsign}`, 300);
						redis.publish('CONTROLLER:UPDATE', controller.vatsimData.callsign);
					} else if (neighbors.includes(controller.artccId)) {
						dataNeighbors.push(controller.artccId);
					}
				}

				for (const atc of redisControllers) {
					if (!dataControllers.includes(atc)) {
						redis.lpush('1231231231231231231231', JSON.stringify(atc), (error) => {
							if (error) {
								console.log('Redis LPUSH error', error);
							}
						});
						redis.publish('CONTROLLER:DELETE', atc);
					}
				}

				redis.set('controllers', dataControllers.join('|'));
				redis.expire('controllers', 65);
				redis.set('neighbors', dataNeighbors.join('|'));
			})
			.catch((err) => console.log('Error doing vNAS datafeed fetch:', err));
	} catch (err) {
		console.log('Error updating controllers:', err);
	}
}

export async function fetchMetars(redis: Redis) {
	try {
		const airportsString = airports.join(','); // Get all METARs, add to database
		fetch(`https://metar.vatsim.net/${airportsString}`, { method: 'GET' })
			.then((response) => response.text())
			.then((data) => {
				const lines = data.split('\n').filter((line) => line.trim() !== '');

				for (const metar of lines) {
					redis.set(`METAR:${metar.slice(0, 4)}`, metar);
					redis.expire(`METAR:${metar.slice(0, 4)}`, 300);
				}
			})
			.catch((err) => console.log('Error doing METAR fetch:', err));
	} catch (err) {
		console.log('Error updating METARs:', err);
	}
}

export async function fetchAtises(redis: Redis) {
	try {
		await PilotOnlineModel.deleteMany({}).exec();

		fetch(`https://data.vatsim.net/v3/vatsim-data.json?t=${Date.now()}`, {
			method: 'GET',
			headers: {
				'Content-Type': 'application/json',
			},
		})
			.then((response) => response.json())
			.then(async (data) => {
				const vatsimData = data as IDataFeed;

				const dataAtis = [];
				const redAtis = await redis.get('atis');
				const redisAtis = redAtis && redAtis.length ? redAtis.split('|') : [];

				for (const atis of vatsimData.atis) {
					const airport = atis.callsign.slice(0, 4);
					if (airports.includes(airport)) {
						dataAtis.push(airport);
						redis.expire(`ATIS:${airport}`, 65);
						redis.publish(`ATIS:${airport}`, atis.atis_code);
					}
				}

				for (const atis of redisAtis) {
					if (!dataAtis.includes(atis)) {
						redis.publish('ATIS:DELETE', atis);
						redis.del(`ATIS:${atis}`);
					}
				}

				redis.set('atis', dataAtis.join('|'));
				redis.expire('atis', 65);
			})
			.catch((err) => console.log('Error doing ATIS fetch', err));
	} catch (err) {
		console.log('Error updating ATISes:', err);
	}
}

export async function fetchPireps() {
	try {
		const now = new Date();
		const twoHours = new Date(now.setHours(now.getHours() - 2));
		await PirepModel.deleteMany({
			$or: [{ manual: false }, { reportTime: { $lte: twoHours } }],
		}).exec();

		fetch('https://aviationweather.gov/api/data/pirep?id=KRFD&distance=200&age=2&format=json', {
			method: 'GET',
			headers: { 'Content-Type': 'application/json' },
		})
			.then((response) => response.json())
			.then((data) => {
				const pireps = data as IPirepFeed[];

				for (const pirep of pireps) {
					if (
						(pirep.pirepType === 'PIREP' || pirep.pirepType === 'Urgent PIREP') &&
						isPointInPolygon([pirep.lon, pirep.lat])
					) {
						const wind = `${pirep.wdir ?? ''}${pirep.wspd ? `@${pirep.wspd}` : ''}`;
						const icing = (
							(pirep.icgInt1 ? pirep.icgInt1 + ' ' : '') + (pirep.icgType1 ? pirep.icgType1 : '')
						)
							.replace(/\s+/g, ' ')
							.trim();
						let skyCond = '';
						if (pirep.clouds && pirep.clouds[0]) {
							skyCond = `${pirep.clouds[0].cover} ${('000' + pirep.clouds[0].base).slice(-3)} ${pirep.clouds[0].top != 0 ? ('000' + pirep.clouds[0].top).slice(-3) : ''}`;
						}
						const turbulence =
							(pirep.tbInt1 ? pirep.tbInt1 + ' ' : '') +
							(pirep.tbFreq1 ? pirep.tbFreq1 + ' ' : '') +
							(pirep.tbType1 ? pirep.tbType1 : '').replace(/\s+/g, ' ').trim();

						PirepModel.create({
							reportTime: new Date(pirep.obsTime * 1000),
							location: pirep.rawOb.slice(0, 3) || '',
							aircraft: pirep.acType || '',
							flightLevel: pirep.fltLvl || 0,
							skyCond: skyCond,
							turbulence: turbulence,
							icing: icing,
							vis: pirep.visib || '',
							temp: pirep.temp || '',
							wind: wind,
							urgent: pirep.pirepType === 'Urgent PIREP',
							raw: pirep.rawOb,
							manual: false,
						});
					}
				}
			})
			.catch((err) => console.log('Error doing PIREP fetch:', err));
	} catch (err) {
		console.log('Error updating PIREPs:', err);
	}
}

function isPointInPolygon(point: [number, number]): boolean {
	const [x, y] = point;

	const numVerticies = airspace.length;
	if (numVerticies < 3) return false; // not a polygon

	let isInside = false;

	for (let i = 0, j = numVerticies - 1; i < numVerticies; j = i++) {
		const [xi, yi] = airspace[i]!;
		const [xj, yj] = airspace[j]!;

		// 1. Check if the point is **ON** the boundary (optional, but good practice)
		// Collinearity check + check if P is within the segment's bounding box.
		if (
			(xj! - x) * (yi! - y) === (xi! - x) * (yj! - y) && // Collinear
			x >= Math.min(xi!, xj!) &&
			x <= Math.max(xi!, xj!) && // Within x-bounds
			y >= Math.min(yi!, yj!) &&
			y <= Math.max(yi!, yj!) // Within y-bounds
		) {
			return true; // Point is on the boundary
		}

		// 2. Ray Casting Algorithm (crossing number)
		// Check if the horizontal ray (y=constant) from the point crosses the edge (i, j).
		if (
			yi! > y !== yj! > y &&
			// Calculate the x-coordinate of the intersection of the ray and the edge.
			// If the point's x is less than the intersection's x, the ray crosses the edge.
			x < ((xj! - xi!) * (y - yi!)) / (yj! - yi!) + xi!
		) {
			isInside = !isInside; // Flip the state (odd = inside, even = outside)
		}
	}

	return isInside;
}
