import "./LeaveReview.css";
import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import Button from "@/modules/core/components/ui/button/Button";
import { CreateReview } from "@/modules/marketplace/models/marketplace";
import { useReviews } from "../../context/ReviewProvider";
import { useWallet } from "@/modules/shared/wallet/WalletProvider";
import StarContainer from "@/modules/shared/star-container/StartContainer";

type FormData = {
  comment: string;
  rating: number;
};

const LeaveReview = () => {
  const { pluginId, addReview, reviewsMap } = useReviews();
  const { isConnected, walletAddress, authToken } = useWallet();
  
  const { watch, setValue, handleSubmit, reset } = useForm<FormData>({
    defaultValues: {
      comment: "",
      rating: 0,
    },
  });

  const [isUpdating, setIsUpdating] = useState(false);

  const comment = watch("comment");
  const rating = watch("rating");

  // Check if user can review based on wallet connection
  const canReview = isConnected && !!authToken && !!walletAddress;

  useEffect(() => {
    // Check if user already has a review and pre-fill the form
    if (walletAddress && reviewsMap?.reviews) {
      const normalized = walletAddress.toLowerCase();
      const existingReview = reviewsMap.reviews.find(
        r => r.address.toLowerCase() === normalized
      );
      if (existingReview) {
        setValue("comment", existingReview.comment);
        setValue("rating", existingReview.rating);
        setIsUpdating(true);
      } else {
        reset();
        setIsUpdating(false);
      }
    }
  }, [reviewsMap, walletAddress, setValue, reset]);

  const onSubmit = (data: FormData) => {
    if (canReview && data.rating && data.comment && walletAddress) {
      // Normalize before sending
      const review: CreateReview = {
        address: walletAddress.toLowerCase(),
        comment: data.comment,
        rating: data.rating,
      };

      addReview(pluginId, review).then((reviewAdded) => {
        if (reviewAdded && !isUpdating) {
          reset();
        }
      });
    }
  };

  return (
    <section className="leave-review">
      <form onSubmit={handleSubmit(onSubmit)}>
        <section className="review-score">
          <label className="label">{isUpdating ? "Update your review" : "Leave a review"}</label>

          <StarContainer
            key={rating}
            initialRating={rating}
            onChange={(newRating) => setValue("rating", newRating)}
          />
        </section>
        <textarea
          cols={78}
          className="review-textarea"
          placeholder={
            canReview
              ? isUpdating
                ? "Update your review here"
                : "Write your review here"
              : "Install the plugin and sign in to leave a review"
          }
          value={comment}
          onChange={(e) => setValue("comment", e.target.value)}
          disabled={!canReview}
        ></textarea>

        <Button
          className={`review-button ${!canReview || !rating || !comment ? "disabled" : ""}`}
          size="medium"
          type="submit"
          styleType="primary"
          disabled={!canReview || !rating || !comment}
        >
          {isUpdating ? "Update review" : "Leave a review"}
        </Button>
      </form>
    </section>
  );
};

export default LeaveReview;
