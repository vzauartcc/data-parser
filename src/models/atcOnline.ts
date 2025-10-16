import { Document, model, Schema } from 'mongoose';
import mongooseLeanVirtuals from 'mongoose-lean-virtuals';

interface IAtcOnline extends Document {
	cid: number;
	name: string;
	rating: number;
	pos: string;
	timeStart: number;
	atis: string;
	frequency: number;

	// Virtuals
	ratingShort: string;
	ratingLong: string;
}

const AtcOnlineSchema = new Schema<IAtcOnline>({
	cid: { type: Number, required: true },
	name: { type: String, required: true },
	rating: { type: Number, required: true },
	pos: { type: String, required: true },
	timeStart: { type: Number, required: true },
	atis: { type: String, required: true, default: '' },
	frequency: { type: Number, required: true },
});

AtcOnlineSchema.plugin(mongooseLeanVirtuals);

export const AtcOnlineModel = model<IAtcOnline>('AtcOnline', AtcOnlineSchema);
