import StarContainer from "@/modules/shared/star-container/StartContainer";
import "./Review.css";

type ReviewProps = {
  id: string;
  date: string;
  rating: number;
  comment: string;
  address?: string;
};

const formatRelativeTime = (isoString: string): string => {
  const date = new Date(isoString);
  const now = new Date();
  const diffInMs = now.getTime() - date.getTime();
  const diffInDays = Math.floor(diffInMs / (1000 * 60 * 60 * 24));
  const diffInHours = Math.floor(diffInMs / (1000 * 60 * 60));
  const diffInMinutes = Math.floor(diffInMs / (1000 * 60));

  if (diffInDays > 0) {
    return `${diffInDays} day${diffInDays > 1 ? 's' : ''} ago`;
  } else if (diffInHours > 0) {
    return `${diffInHours} hour${diffInHours > 1 ? 's' : ''} ago`;
  } else if (diffInMinutes > 0) {
    return `${diffInMinutes} minute${diffInMinutes > 1 ? 's' : ''} ago`;
  } else {
    return 'Just now';
  }
};

const truncateAddress = (address: string): string => {
  if (!address || address.length <= 8) return address;
  return `${address.slice(0, 4)}...${address.slice(-4)}`;
};

const Review = ({ id, date, rating, comment, address }: ReviewProps) => {
  return (
    <div className="single-review">
      <div className="review-info-header" key={id}>
        <div className="review-user-info">
          <div className="review-icon"></div>
          <div className="review-user-details">
            {address && (
              <div className="review-address">{truncateAddress(address)}</div>
            )}
            <div className="review-date">{formatRelativeTime(date)}</div>
          </div>
        </div>
        <StarContainer initialRating={rating} disableChange={true} />
      </div>
      <div>{`"${comment}"`}</div>
    </div>
  );
};

export default Review;
