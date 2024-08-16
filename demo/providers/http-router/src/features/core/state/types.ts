import type {StateCreator} from 'zustand';
import type {RootStore} from './store';

type ImmerStateCreator<T> = StateCreator<RootStore, [['zustand/immer', never], never], [], T>;

type RootStateCreator = ImmerStateCreator<RootStore>;

type SliceCreator<TSlice extends keyof RootStore> = (
  ...params: Parameters<RootStateCreator>
) => Pick<ReturnType<RootStateCreator>, TSlice>;

export type {ImmerStateCreator, SliceCreator};
