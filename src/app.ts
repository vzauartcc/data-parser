import { Cron } from 'croner';
import { Redis } from 'ioredis';
import mongoose from 'mongoose';
import { fetchAtises, fetchControllers, fetchMetars, fetchPilots, fetchPireps } from './feed.js';

if (
	!process.env['MONGO_URI'] ||
	!process.env['REDIS_URI'] ||
	!process.env['ZAU_API_URL'] ||
	!process.env['ZAU_API_KEY']
) {
	console.error(
		'Missing at least one environment variable. Check to make sure the follow are set: "MONGO_URI", "REDIS_URI", "ZAU_API_URL", "ZAU_API_KEY"',
	);
	process.exit(4);
}

const redis = new Redis(process.env['REDIS_URI']);
redis.on('connect', () => console.log('Connected to redis'));
redis.on('error', (err) => {
	throw err;
});

mongoose.set('toJSON', { virtuals: true });
mongoose.connect(process.env['MONGO_URI']);

mongoose.connection.once('open', () => {
	console.log(`Connected to database, scheduling next task run. . . .`);

	// 15 second updates
	new Cron('*/15 * * * * *', () => doWork());

	// 2 minute updates
	new Cron('*/2 * * * *', () => fetchPireps());
});

function doWork() {
	fetchPilots(redis);
	fetchControllers(redis);
	fetchMetars(redis);
	fetchAtises(redis);
}
