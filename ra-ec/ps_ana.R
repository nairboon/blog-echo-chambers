require(geoR)
require(lattice)
data(elevation)

pfU <- read.csv("~/dev/go/src/flache/ra-ec/t.csv")

n = length(pfU$mu)
#grid start x/y
gsx=min(pfU$mu)
gsy=min(pfU$ponline)

#grid end x/y
gex=max(pfU$mu)
gey=max(pfU$ponline)

sample=0.01


 elevation.df = data.frame(x = pfU$mu,
                           y = pfU$ponline, z = pfU$deltares)
# 
# elevation.loess = loess(z ~ x*y, data = elevation.df,
#                         degree = 2, span = 0.25)
# 
# elevation.fit = expand.grid(list(x = seq(gsx, gex, sample), y = seq(gsy, gey, sample)))
# 
# z = predict(elevation.loess, newdata = elevation.fit)

#View(z)
# elevation.fit$Height = as.numeric(z)
# 
# 
# 
# image(seq(gsx, gex, sample), seq(gsy, gey, sample), z,
#       xlab = "mu", ylab = "ponline",
#       main = paste("Sample of" , as.character(n)))
# #par(new=T)
# points(pfU$mu,pfU$ponline)
# box()


# p <- ggplot(elevation.fit, aes(x, y, fill = Height)) + geom_tile() +
#   xlab("X Coordinate (feet)") + ylab("Y Coordinate (feet)") +
#   labs(title = "Surface elevation data") +
#   scale_fill_gradient(limits = c(0, 1),low = "black",high = "white") +
#   scale_x_continuous(expand = c(0,0)) +
#   scale_y_continuous(expand = c(0,0))

# print(p)
library(ggplot2)


alpha <- ggplot(elevation.df, aes(x = pfU$mu, y = pfU$deltares)) + geom_point() +
#    stat_smooth(method = 'lm', aes(colour = 'linear'), se = FALSE) +
   stat_smooth(method = 'lm', formula = y ~ poly(x,4), aes(colour = 'polynomial'), se= FALSE) +
#    stat_smooth(method = 'nls', formula = y ~ a * log(x) +b, aes(colour = 'logarithmic'), se = FALSE, start = list(a=1,b=1)) +
#    stat_smooth(method = 'nls', formula = y ~ a*exp(b *x), aes(colour = 'Exponential'), se = FALSE, start = list(a=1,b=1)) +
  theme_bw() + ylab("Ratio of Echo chambers")+
 xlab("µ") +
  scale_colour_brewer(name = 'Trendline', palette = 'Set2')

print(alpha)
