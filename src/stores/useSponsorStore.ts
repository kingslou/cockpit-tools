import { create } from 'zustand';
import buildConfig from '../../build-config.json';
import type { SponsorModuleState } from '../types/sponsor';
import {
  forceRefreshSponsorModuleState,
  getSponsorModuleState,
  syncSponsorRoutes,
} from '../services/sponsorService';

const EMPTY_STATE: SponsorModuleState = {
  sponsorModule: null,
};

interface SponsorStoreState {
  state: SponsorModuleState;
  loading: boolean;
  initialized: boolean;
  fetchState: (force?: boolean) => Promise<SponsorModuleState>;
}

export const useSponsorStore = create<SponsorStoreState>((set) => ({
  state: EMPTY_STATE,
  loading: false,
  initialized: false,

  fetchState: async (force = false) => {
    // 本地发行配置优先，远端推荐和线路同步不能重新打开推广。
    if (!buildConfig.promotionsEnabled) {
      set({ state: EMPTY_STATE, loading: false, initialized: true });
      return EMPTY_STATE;
    }
    set({ loading: true });
    try {
      const nextState = force
        ? await forceRefreshSponsorModuleState()
        : await getSponsorModuleState();
      set({ state: nextState, loading: false, initialized: true });
      if (nextState.sponsorModule) {
        void syncSponsorRoutes()
          .then((summary) => {
            if (summary.changed) {
              window.dispatchEvent(new CustomEvent('sponsor-routes-updated'));
            }
          })
          .catch((error) => {
            console.warn('同步赞助商线路失败:', error);
          });
      }
      return nextState;
    } catch (error) {
      console.error('加载赞助商模块失败:', error);
      set({ state: EMPTY_STATE, loading: false, initialized: true });
      return EMPTY_STATE;
    }
  },
}));
