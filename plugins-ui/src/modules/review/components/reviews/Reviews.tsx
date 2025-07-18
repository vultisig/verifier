import LeaveReview from "@/modules/review/components/leave-review/LeaveReview";
import ReviewHistory from "@/modules/review/components/review-history/ReviewHistory";
import "./Reviews.css";

import Rating from "@/modules/shared/rating/Rating";

const Reviews = () => {
  return (
    <>
      <section>
        <h3 className="review-rating-header">Reviews and Ratings</h3>
        <div className="review-rating">
          <LeaveReview />
          <Rating />
        </div>
      </section>

      <section className="reviews">
        <ReviewHistory />
      </section>
    </>
  );
};

export default Reviews;
