import "./LeaveReview.css";
import { useState, useEffect } from "react";
import Button from "@/modules/core/components/ui/button/Button";
import { CreateReview } from "@/modules/marketplace/models/marketplace";
import { useReviews } from "../../context/ReviewProvider";
import StarContainer from "@/modules/shared/star-container/StartContainer";

const LeaveReview = () => {
  const { pluginId, addReview } = useReviews();

  const [input, setInput] = useState("");
  const [rating, setRating] = useState(0);
  const [canReview, setCanReview] = useState(false);

  useEffect(() => {
    setCanReview(!!localStorage.getItem("authToken"));
  }, []);

  const submitReview = () => {
    if (canReview && rating && input) {
      const review: CreateReview = {
        address: localStorage.getItem("walletAddress") || "",
        comment: input,
        rating: rating,
      };

      addReview(pluginId, review).then((reviewAdded) => {
        if (reviewAdded) {
          setInput("");
          setRating(0);
        }
      });
    }
  };

  return (
    <section className="leave-review">
      <section className="review-score">
        <label className="label">Leave a review</label>

        <StarContainer
          key={rating}
          initialRating={rating}
          onChange={setRating}
        />
      </section>
      <textarea
        cols={78}
        className="review-textarea"
        placeholder={
          canReview
            ? "Write your review here"
            : "Install the plugin and sign in to leave a review"
        }
        value={input}
        onChange={(e) => setInput(e.target.value)}
        disabled={!canReview}
      ></textarea>

      <Button
        className={`review-button ${!canReview || !rating || !input ? "disabled" : ""}`}
        size="medium"
        type="button"
        styleType="primary"
        onClick={submitReview}
        disabled={!canReview || !rating || !input}
      >
        Leave a review
      </Button>
    </section>
  );
};

export default LeaveReview;
