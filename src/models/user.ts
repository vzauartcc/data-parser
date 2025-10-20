import { Document, model, Schema } from 'mongoose';

export interface ICertificationDate {
	code: string;
	gainedDate: Date;
}

export interface IUser extends Document {
	cid: number;
	fname: string;
	lname: string;
}

const UserSchema = new Schema<IUser>(
	{
		cid: { type: Number, required: true, unique: true },
		fname: { type: String, required: true },
		lname: { type: String, required: true },
	},
	{
		timestamps: true,
	},
);

export const UserModel = model<IUser>('User', UserSchema);
