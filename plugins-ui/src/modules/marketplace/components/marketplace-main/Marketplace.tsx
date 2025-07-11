import PluginCard from "@/modules/plugin/components/plugin-card/PluginCard";
import "./Marketplace.css";
import MarketplaceFilters from "../marketplace-filters/MarketplaceFilters";
import { PluginFilters } from "../marketplace-filters/MarketplaceFilters";
import { useEffect, useState } from "react";
import { PluginMap, ViewFilter } from "../../models/marketplace";
import { Category } from "../../models/category";
import MarketplaceService from "../../services/marketplaceService";
import Pagination from "@/modules/core/components/ui/pagination/Pagination";
import { publish } from "@/utils/eventBus";

const getSavedView = (): string => {
  return localStorage.getItem("view") || "grid";
};

const getCategoryName = (categories: Category[], id: string) => {
  const category = categories.find((c) => c.id === id);
  if (!category) return "";

  return category.name;
};

const ITEMS_PER_PAGE = 6;
const DEBOUNCE_DELAY = 500;

const Marketplace = () => {
  const [view, setView] = useState<string>(getSavedView());

  const [currentPage, setCurrentPage] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [filters, setFilters] = useState<PluginFilters>({
    term: "",
    categoryId: "",
    sortBy: "created_at",
    sortOrder: "DESC",
  });
  const [categories, setCategories] = useState<Category[]>([]);

  const changeView = (view: ViewFilter) => {
    localStorage.setItem("view", view);
    setView(view);
  };

  const [pluginsMap, setPlugins] = useState<PluginMap | null>(null);

  useEffect(() => {
    const fetchCategories = async (): Promise<void> => {
      try {
        const fetchedCategories = await MarketplaceService.getCategories();
        setCategories(fetchedCategories);
      } catch (error) {
        if (error instanceof Error) {
          console.error("Failed to get categories:", error.message);
          publish("onToast", {
            message: error.message || "Failed to get categories",
            type: "error",
          });
        }
      }
    };

    fetchCategories();
  }, []);

  useEffect(() => {
    const fetchPlugins = async (): Promise<void> => {
      try {
        const fetchedPlugins = await MarketplaceService.getPlugins();
        console.log("here fetchedPlugins");
        console.log(fetchedPlugins);
        setPlugins(fetchedPlugins);
        setTotalPages(
          Math.ceil(
            fetchedPlugins.total_count
              ? fetchedPlugins.total_count / ITEMS_PER_PAGE
              : 1
          )
        );

        if (
          fetchedPlugins &&
          fetchedPlugins.total_count / ITEMS_PER_PAGE > 1 &&
          currentPage === 0
        ) {
          setCurrentPage(1);
        }
      } catch (error) {
        if (error instanceof Error) {
          console.error("Failed to get plugins:", error.message);
          publish("onToast", {
            type: "error",
            message: "Failed to get plugins",
          });
        }
      }
    };

    const timeout = setTimeout(() => {
      fetchPlugins();
    }, DEBOUNCE_DELAY);

    return () => clearTimeout(timeout);
  }, [filters, currentPage]);

  const onCurrentPageChange = (page: number): void => {
    setCurrentPage(page);
  };

  return (
    <>
      {categories && categories.length > 0 && pluginsMap && (
        <div className="only-section">
          <h2>Plugins Marketplace</h2>
          <MarketplaceFilters
            categories={categories}
            viewFilter={view as ViewFilter}
            onViewChange={changeView}
            filters={filters}
            onFiltersChange={setFilters}
          />
          <section className="cards">
            {pluginsMap.plugins?.map((plugin) => (
              <div
                className={view === "list" ? "list-card" : ""}
                key={plugin.id}
              >
                <PluginCard
                  uiStyle={view as ViewFilter}
                  id={plugin.id}
                  title={plugin.title}
                  description={plugin.description}
                  categoryName={getCategoryName(categories, plugin.category_id)}
                />
              </div>
            ))}
          </section>

          {totalPages > 1 && (
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={onCurrentPageChange}
            />
          )}
        </div>
      )}
    </>
  );
};

export default Marketplace;
