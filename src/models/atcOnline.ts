/**
 * Any changes to this model should be reflected to both the api and data-parser repositories.
 **/

import { Document, model, Schema } from 'mongoose';
import mongooseLeanVirtuals from 'mongoose-lean-virtuals';

interface IAtcOnline extends Document {
	cid: number;
	name: string;
	rating: number;
	pos: string;
	timeStart: Date;
	atis: string;
	frequency: number;
}

const AtcOnlineSchema = new Schema<IAtcOnline>(
	{
		cid: { type: Number, required: true },
		name: { type: String, required: true },
		rating: { type: Number, required: true },
		pos: { type: String, required: true },
		timeStart: { type: Date, required: true },
		atis: { type: String, required: true, default: '' },
		frequency: { type: Number, required: true },
	},
	{ collection: 'atcOnline' },
);

AtcOnlineSchema.plugin(mongooseLeanVirtuals);

export const AtcOnlineModel = model<IAtcOnline>('AtcOnline', AtcOnlineSchema);
