# CEC 2005 数据与实现说明

本目录包含 CEC 2005 F1、F2、F9 的 100 维移位向量，运行时按所选维数取前 D 项。
数据来自 opfunu 的 CEC 2005 数据镜像，保留其 GPL-3.0 许可证于 LICENSE.txt。
来源：https://github.com/thieu1995/opfunu/tree/master/opfunu/cec_based/data_2005
获取日期：2026-09-14。

定义参考：Suganthan, P. N. et al. (2005), Problem Definitions and Evaluation Criteria
for the CEC 2005 Special Session on Real-Parameter Optimization, KanGAL Report 2005005。

本项目自行用 Go 实现数学定义：F1 为移位平方和，F2 为包含全部 D 项的前缀和平方和，
F9 为移位 Rastrigin。F1/F2 偏置 -450，范围 [-100,100]；F9 偏置 -330，范围 [-5,5]。
二维用于教学展示；10、30、50 维适用于课程对比。该模块是所列三个函数的实验子集，
不是完整的 25 函数竞赛评测。热力图将第 3～D 维固定在移位最优点，仅前两维变化；
高维种群点是投影，其函数值不等于该点热力图切片的高度。
