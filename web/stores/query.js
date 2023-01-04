import create from "zustand";

const useQuery = create((set) => ({
  search: "",
  sort: "latest",
  nsfw: "exclude",
  bw: "exclude",
  sprocket: "include",
  setSearch: (search) => set({ search }),
  setSort: (sort) => set({ sort }),
  setNsfw: (nsfw) => set({ nsfw }),
  setBw: (bw) => set({ bw }),
  setSprocket: (sprocket) => set({ sprocket }),
}));

export default useQuery;
