import m from 'mongoose';
// import ControllerHours from './ControllerHours.js';

const userSchema = new m.Schema({
	cid: Number,
	fname: String,
	lname: String,
	email: String,
	rating: Number,
	oi: String,
	member: Boolean,
	vis: Boolean,
	homeFacility: String,
});

export default m.model('User', userSchema);